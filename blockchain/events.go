package blockchain

import (
	"context"
	"fmt"

	"time"

	"math/big"

	"strings"

	"bytes"

	"github.com/SmartMeshFoundation/Atmosphere/contracts"
	"github.com/SmartMeshFoundation/Atmosphere/log"
	"github.com/SmartMeshFoundation/Atmosphere/network/helper"
	"github.com/SmartMeshFoundation/Atmosphere/network/rpc"
	"github.com/SmartMeshFoundation/Atmosphere/transfer"
	"github.com/SmartMeshFoundation/Atmosphere/transfer/mediatedtransfer"
	"github.com/SmartMeshFoundation/Atmosphere/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

var secretRegistryAbi abi.ABI
var tokenNetworkAbi abi.ABI
var topicToEventName map[common.Hash]string

func init() {
	var err error
	secretRegistryAbi, err = abi.JSON(strings.NewReader(contracts.SecretRegistryABI))
	if err != nil {
		panic(fmt.Sprintf("secretRegistryAbi parse err %s", err))
	}
	tokenNetworkAbi, err = abi.JSON(strings.NewReader(contracts.TokenNetworkABI))
	if err != nil {
		panic(fmt.Sprintf("tokenNetworkAbi parse err %s", err))
	}
	topicToEventName = make(map[common.Hash]string)
	topicToEventName[tokenNetworkAbi.Events[nameChannelOpenedAndDeposit].Id()] = nameChannelOpenedAndDeposit
	topicToEventName[tokenNetworkAbi.Events[nameChannelNewDeposit].Id()] = nameChannelNewDeposit
	topicToEventName[tokenNetworkAbi.Events[nameChannelClosed].Id()] = nameChannelClosed
	topicToEventName[tokenNetworkAbi.Events[nameChannelUnlocked].Id()] = nameChannelUnlocked
	topicToEventName[tokenNetworkAbi.Events[nameBalanceProofUpdated].Id()] = nameBalanceProofUpdated
	topicToEventName[tokenNetworkAbi.Events[nameChannelPunished].Id()] = nameChannelPunished
	topicToEventName[tokenNetworkAbi.Events[nameChannelSettled].Id()] = nameChannelSettled
	topicToEventName[tokenNetworkAbi.Events[nameChannelCooperativeSettled].Id()] = nameChannelCooperativeSettled
	topicToEventName[tokenNetworkAbi.Events[nameChannelWithdraw].Id()] = nameChannelWithdraw
	topicToEventName[secretRegistryAbi.Events[nameSecretRevealed].Id()] = nameSecretRevealed

}

/*
Events handles all contract events from blockchain
*/
type Events struct {
	StateChangeChannel chan transfer.StateChange
	Config             *config

	lastBlockNumber     int64
	rpcModuleDependency RPCModuleDependency
	client              *helper.SafeEthClient
	stopChan            chan int               // has stopped?
	txDone              map[common.Hash]uint64 // 该map记录最近30块内处理的events流水,用于事件去重
}

//NewBlockChainEvents create BlockChainEvents
func NewBlockChainEvents(client *helper.SafeEthClient, rpcModuleDependency RPCModuleDependency) *Events {
	be := &Events{
		StateChangeChannel:  make(chan transfer.StateChange, 10),
		rpcModuleDependency: rpcModuleDependency,
		client:              client,
		txDone:              make(map[common.Hash]uint64),
		Config:              newDefaultConfig(),
	}
	return be
}

//Stop event listenging
func (be *Events) Stop() {
	if be.stopChan != nil {
		close(be.stopChan)
	}
	log.Info("Events stop ok...")
}

/*
Start listening events send to  channel can duplicate but cannot lose.
1. first resend events may lost (duplicat is ok)
2. listen new events on blockchain
有可能启动的时候没联网,等到启动以后某个事件连上了以后在处理.
1.要保证事件按照顺序抵达
2. 保证事件不丢失
3. 事件是可以重复的
*/
/*
 *  Start listening events send to channel can duplicate but cannot lose.
 *  1. first resend events may lost (duplicate is ok)
 *  2. listen new events on blockchain
 *
 *  It is possible that there is no internet connection when start-up, and missed events have to be regained
 *  after those events starts.
 * 	1. Make sure events sending out with order
 *  2. Make sure events does not get lost.
 *  3. Make sure repeated events are allowed.
 */
func (be *Events) Start(LastBlockNumber int64) {
	log.Info(fmt.Sprintf("get state change since %d", LastBlockNumber))
	be.lastBlockNumber = LastBlockNumber
	/*
		1. start alarm task
	*/
	go be.startAlarmTask()
}

func (be *Events) startAlarmTask() {
	log.Trace(fmt.Sprintf("start getting lasted block number from blocknubmer=%d", be.lastBlockNumber))
	startUpBlockNumber := be.lastBlockNumber
	currentBlock := be.lastBlockNumber
	retryTime := 0
	be.stopChan = make(chan int)
	for {
		ctx, cancelFunc := context.WithTimeout(context.Background(), be.Config.RPCTimeOut)
		h, err := be.client.HeaderByNumber(ctx, nil)
		if err != nil {
			log.Error(fmt.Sprintf("HeaderByNumber err=%s", err))
			cancelFunc()
			if be.stopChan != nil {
				go be.client.RecoverDisconnect()
			}
			return
		}
		cancelFunc()
		lastedBlock := h.Number.Int64()
		if currentBlock >= lastedBlock {
			if startUpBlockNumber == lastedBlock {
				// 当启动时获取不到新块,也需要通知photonService,否则会导致api无法启动
				log.Warn(fmt.Sprintf("atmosphere start with blockNumber %d,but lastedBlockNumber on chain also %d", startUpBlockNumber, lastedBlock))
				be.StateChangeChannel <- &transfer.BlockStateChange{BlockNumber: currentBlock}
				startUpBlockNumber = 0
			}
			time.Sleep(be.Config.RPCPollPeriod / 2)
			retryTime++
			if retryTime > 10 {
				log.Warn(fmt.Sprintf("get same block number %d from chain %d times,maybe something wrong with smc ...", lastedBlock, retryTime))
			}
			continue
		}
		retryTime = 0
		if currentBlock != -1 && lastedBlock != currentBlock+1 {
			log.Warn(fmt.Sprintf("AlarmTask missed %d blocks", lastedBlock-currentBlock-1))
		}
		if lastedBlock%be.Config.LogPeriod == 0 {
			log.Trace(fmt.Sprintf("new block :%d", lastedBlock))
		}

		fromBlockNumber := currentBlock - 2*be.Config.ForkConfirmNumber
		if fromBlockNumber < 0 {
			fromBlockNumber = 0
		}
		// get all state change between currentBlock and lastedBlock
		stateChanges, err := be.queryAllStateChange(fromBlockNumber, lastedBlock)
		if err != nil {
			log.Error(fmt.Sprintf("queryAllStateChange err=%s", err))
			// 如果这里出现err,不能继续处理该blocknumber,否则会丢事件,直接从该块重新处理即可
			time.Sleep(be.Config.RPCPollPeriod / 2)
			continue
		}
		if len(stateChanges) > 0 {
			log.Trace(fmt.Sprintf("receive %d events between block %d - %d", len(stateChanges), currentBlock+1, lastedBlock))
		}

		// refresh block number and notify PhotonService
		currentBlock = lastedBlock
		be.lastBlockNumber = currentBlock

		be.StateChangeChannel <- &transfer.BlockStateChange{BlockNumber: currentBlock}

		// notify Atmosphere service
		for _, sc := range stateChanges {
			be.StateChangeChannel <- sc
		}

		// 清除过期流水
		for key, blockNumber := range be.txDone {
			if blockNumber <= uint64(fromBlockNumber) {
				delete(be.txDone, key)
			}
		}
		// wait to next time
		select {
		case <-time.After(be.Config.RPCPollPeriod):
		case <-be.stopChan:
			be.stopChan = nil
			log.Info(fmt.Sprintf("AlarmTask quit complete"))
			return
		}
	}
}

func (be *Events) queryAllStateChange(fromBlock int64, toBlock int64) (stateChanges []mediatedtransfer.ContractStateChange, err error) {
	/*
		get all event of contract TokenNetworkRegistry, SecretRegistry , TokenNetwork
	*/
	logs, err := be.getLogsFromChain(fromBlock, toBlock)
	if err != nil {
		return
	}
	stateChanges, err = be.parseLogsToEvents(logs)
	if err != nil {
		return
	}
	// 排序
	sortContractStateChange(stateChanges)
	return
}

func (be *Events) getLogsFromChain(fromBlock int64, toBlock int64) (logs []types.Log, err error) {
	/*
		get all event of contract TokenNetworkRegistry, SecretRegistry , TokenNetwork
	*/
	contractAddresses := []common.Address{
		be.rpcModuleDependency.GetTokenNetworkAddress(),
		be.rpcModuleDependency.GetSecretRegistryAddress(),
	}
	logs, err = rpc.EventsGetInternal(
		rpc.GetQueryConext(), contractAddresses, fromBlock, toBlock, be.client)
	if err != nil {
		return
	}
	return
}

func (be *Events) parseLogsToEvents(logs []types.Log) (stateChanges []mediatedtransfer.ContractStateChange, err error) {
	for _, l := range logs {
		eventName := topicToEventName[l.Topics[0]]

		// 根据已处理流水去重
		if doneBlockNumber, ok := be.txDone[l.TxHash]; ok {
			if doneBlockNumber == l.BlockNumber {
				//log.Trace(fmt.Sprintf("get event txhash=%s repeated,ignore...", l.TxHash.String()))
				continue
			}
			log.Warn(fmt.Sprintf("event tx=%s happened at %d, but now happend at %d ", l.TxHash.String(), doneBlockNumber, l.BlockNumber))
		}

		// open,deposit,withdraw事件延迟确认,开关默认关闭,方便测试
		if be.Config.EnableForkConfirm && needConfirm(eventName) {
			if be.lastBlockNumber-int64(l.BlockNumber) < be.Config.ForkConfirmNumber {
				continue
			}
			log.Info(fmt.Sprintf("event %s tx=%s happened at %d, confirmed at %d", eventName, l.TxHash.String(), l.BlockNumber, be.lastBlockNumber))
		}
		// registry secret事件延迟确认,否则在出现恶意分叉的情况下,中间节点有损失资金的风险
		if eventName == nameSecretRevealed {
			if be.lastBlockNumber-int64(l.BlockNumber) < be.Config.ForkConfirmNumber {
				continue
			}
			log.Info(fmt.Sprintf("event %s tx=%s happened at %d, confirmed at %d", eventName, l.TxHash.String(), l.BlockNumber, be.lastBlockNumber))
		}

		switch eventName {
		case nameChannelOpenedAndDeposit:
			e, err2 := newEventChannelOpenAndDeposit(&l)
			if err = err2; err != nil {
				return
			}
			oev, dev := eventChannelOpenAndDeposit2StateChange(e)
			stateChanges = append(stateChanges, oev)
			stateChanges = append(stateChanges, dev)
		case nameChannelNewDeposit:
			e, err2 := newEventChannelNewDeposit(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelNewDeposit2StateChange(e))
		case nameChannelClosed:
			e, err2 := newEventChannelClosed(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelClosed2StateChange(e))
		case nameChannelUnlocked:
			e, err2 := newEventChannelUnlocked(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelUnlocked2StateChange(e))
		case nameBalanceProofUpdated:
			e, err2 := newEventBalanceProofUpdated(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventBalanceProofUpdated2StateChange(e))
		case nameChannelPunished:
			e, err2 := newEventChannelPunished(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelPunished2StateChange(e))
		case nameChannelSettled:
			e, err2 := newEventChannelSettled(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelSettled2StateChange(e))
		case nameChannelCooperativeSettled:
			e, err2 := newEventChannelCooperativeSettled(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelCooperativeSettled2StateChange(e))
		case nameChannelWithdraw:
			e, err2 := newEventChannelWithdraw(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventChannelWithdraw2StateChange(e))
		case nameSecretRevealed:
			e, err2 := newEventSecretRevealed(&l)
			if err = err2; err != nil {
				return
			}
			stateChanges = append(stateChanges, eventSecretRevealed2StateChange(e))
		default:
			log.Warn(fmt.Sprintf("receive unkonwn type event from chain : \n%s\n", utils.StringInterface(l, 3)))
		}
		// 记录处理流水
		be.txDone[l.TxHash] = l.BlockNumber
	}
	return
}

func needConfirm(eventName string) bool {

	if eventName == nameChannelOpenedAndDeposit ||
		eventName == nameChannelNewDeposit ||
		eventName == nameChannelWithdraw {
		return true
	}
	return false
}

//eventChannelOpenAndDeposit2StateChange to state change
func eventChannelOpenAndDeposit2StateChange(ev *contracts.TokenNetworkChannelOpenedAndDeposit) (ch1 *mediatedtransfer.ContractNewChannelStateChange, ch2 *mediatedtransfer.ContractBalanceStateChange) {
	channelIdentifier := calcChannelID(ev.Token, ev.Participant, ev.Partner)
	ch1 = &mediatedtransfer.ContractNewChannelStateChange{
		ChannelIdentifier: &contracts.ChannelUniqueID{
			ChannelIdentifier: channelIdentifier,
			OpenBlockNumber:   int64(ev.Raw.BlockNumber),
		},
		TokenAddress:  ev.Token,
		Participant1:  ev.Participant,
		Participant2:  ev.Partner,
		SettleTimeout: int(ev.SettleTimeout),
		BlockNumber:   int64(ev.Raw.BlockNumber),
	}
	ch2 = &mediatedtransfer.ContractBalanceStateChange{
		ChannelIdentifier:  channelIdentifier,
		ParticipantAddress: ev.Participant,
		Balance:            ev.Participant1Deposit,
		BlockNumber:        int64(ev.Raw.BlockNumber),
	}
	return
}

//注意与合约上计算方式保持完全一致.
func calcChannelID(token, p1, p2 common.Address) common.Hash {
	var channelID common.Hash
	if bytes.Compare(p1[:], p2[:]) < 0 {
		channelID = utils.Sha3(p1[:], p2[:], token[:])
	} else {
		channelID = utils.Sha3(p2[:], p1[:], token[:])
	}
	return channelID
}

//eventChannelNewDeposit2StateChange to statechange
func eventChannelNewDeposit2StateChange(ev *contracts.TokenNetworkChannelNewDeposit) *mediatedtransfer.ContractBalanceStateChange {
	return &mediatedtransfer.ContractBalanceStateChange{
		ChannelIdentifier:  ev.ChannelIdentifier,
		ParticipantAddress: ev.Participant,
		Balance:            ev.TotalDeposit,
		BlockNumber:        int64(ev.Raw.BlockNumber),
	}
}

//eventChannelClosed2StateChange to statechange
func eventChannelClosed2StateChange(ev *contracts.TokenNetworkChannelClosed) *mediatedtransfer.ContractClosedStateChange {
	c := &mediatedtransfer.ContractClosedStateChange{
		ChannelIdentifier: ev.ChannelIdentifier,
		ClosingAddress:    ev.ClosingParticipant,
		LocksRoot:         ev.Locksroot,
		TransferredAmount: ev.TransferredAmount,
		ClosedBlock:       int64(ev.Raw.BlockNumber),
	}
	if ev.TransferredAmount == nil {
		c.TransferredAmount = new(big.Int)
	}
	return c
}

//eventChannelUnlocked2StateChange to statechange
func eventChannelUnlocked2StateChange(ev *contracts.TokenNetworkChannelUnlocked) *mediatedtransfer.ContractUnlockStateChange {
	c := &mediatedtransfer.ContractUnlockStateChange{
		ChannelIdentifier: ev.ChannelIdentifier,
		Participant:       ev.PayerParticipant,
		LockHash:          ev.Lockhash,
		TransferAmount:    ev.TransferredAmount,
		BlockNumber:       int64(ev.Raw.BlockNumber),
	}
	if c.TransferAmount == nil {
		c.TransferAmount = new(big.Int)
	}
	return c
}

//eventBalanceProofUpdated2StateChange to statechange
func eventBalanceProofUpdated2StateChange(ev *contracts.TokenNetworkBalanceProofUpdated) *mediatedtransfer.ContractBalanceProofUpdatedStateChange {
	c := &mediatedtransfer.ContractBalanceProofUpdatedStateChange{
		ChannelIdentifier: ev.ChannelIdentifier,
		Participant:       ev.Participant,
		LocksRoot:         ev.Locksroot,
		TransferAmount:    ev.TransferredAmount,
		BlockNumber:       int64(ev.Raw.BlockNumber),
	}
	if c.TransferAmount == nil {
		c.TransferAmount = new(big.Int)
	}
	return c
}

//eventChannelPunished2StateChange to stateChange
func eventChannelPunished2StateChange(ev *contracts.TokenNetworkChannelPunished) *mediatedtransfer.ContractPunishedStateChange {
	return &mediatedtransfer.ContractPunishedStateChange{
		ChannelIdentifier: common.Hash(ev.ChannelIdentifier),
		Beneficiary:       ev.Beneficiary,
		BlockNumber:       int64(ev.Raw.BlockNumber),
	}
}

//eventChannelSettled2StateChange to stateChange
func eventChannelSettled2StateChange(ev *contracts.TokenNetworkChannelSettled) *mediatedtransfer.ContractSettledStateChange {
	return &mediatedtransfer.ContractSettledStateChange{
		ChannelIdentifier:  common.Hash(ev.ChannelIdentifier),
		Participant1Amount: ev.Participant1Amount,
		Participant2Amount: ev.Participant2Amount,
		SettledBlock:       int64(ev.Raw.BlockNumber),
	}
}

//eventChannelCooperativeSettled2StateChange to stateChange
func eventChannelCooperativeSettled2StateChange(ev *contracts.TokenNetworkChannelCooperativeSettled) *mediatedtransfer.ContractCooperativeSettledStateChange {
	return &mediatedtransfer.ContractCooperativeSettledStateChange{
		ChannelIdentifier:  common.Hash(ev.ChannelIdentifier),
		Participant1Amount: ev.Participant1Amount,
		Participant2Amount: ev.Participant2Amount,
		SettledBlock:       int64(ev.Raw.BlockNumber),
	}
}

//eventChannelWithdraw2StateChange to stateChange
func eventChannelWithdraw2StateChange(ev *contracts.TokenNetworkChannelWithdraw) *mediatedtransfer.ContractChannelWithdrawStateChange {
	c := &mediatedtransfer.ContractChannelWithdrawStateChange{
		ChannelIdentifier: &contracts.ChannelUniqueID{
			ChannelIdentifier: common.Hash(ev.ChannelIdentifier),
			OpenBlockNumber:   int64(ev.Raw.BlockNumber),
		},
		Participant1:        ev.Participant1,
		Participant1Balance: ev.Participant1Balance,
		Participant2:        ev.Participant2,
		Participant2Balance: ev.Participant2Balance,
		BlockNumber:         int64(ev.Raw.BlockNumber),
	}
	if c.Participant1Balance == nil {
		c.Participant1Balance = new(big.Int)
	}
	if c.Participant2Balance == nil {
		c.Participant2Balance = new(big.Int)
	}
	return c
}

//eventSecretRevealed2StateChange to statechange
func eventSecretRevealed2StateChange(ev *contracts.SecretRegistrySecretRevealed) *mediatedtransfer.ContractSecretRevealOnChainStateChange {
	return &mediatedtransfer.ContractSecretRevealOnChainStateChange{
		Secret:         ev.Secret,
		LockSecretHash: utils.ShaSecret(ev.Secret[:]),
		BlockNumber:    int64(ev.Raw.BlockNumber),
	}
}
