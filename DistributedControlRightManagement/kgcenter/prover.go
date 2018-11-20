package kgcenter

import (
	"math/big"
)

type ProverInfo struct {
	xShare, xShareRnd, encXShare *big.Int
	yShare_x                     *big.Int
	yShare_y                     *big.Int

	mpkEncXiYi  *MultiTrapdoorCommitment
	openEncXiYi *Open
	cmtEncXiYi  *Commitment

	zkpKG *Zkp
	encX  *big.Int

	pk_x *big.Int
	pk_y *big.Int

	rhoI, rhoIRnd, uI, vI *big.Int
	mpkUiVi               *MultiTrapdoorCommitment
	openUiVi              *Open
	cmtUiVi               *Commitment

	zkp1          *Zkpi1
	kI, cI, cIRnd *big.Int
	rI_x          *big.Int
	rI_y          *big.Int

	mask, wI *big.Int
	mpkRiWi  *MultiTrapdoorCommitment
	openRiWi *Open
	cmtRiWi  *Commitment

	zkp_i2 *Zkpi2
}

func (pi *ProverInfo) getxShare() *big.Int {
	return pi.xShare
}

func (pi *ProverInfo) setxShare(xShare *big.Int) {
	pi.xShare = xShare //设置的就是私钥片
}

func (pi *ProverInfo) getxShareRnd() *big.Int {
	return pi.xShareRnd
}

func (pi *ProverInfo) setxShareRnd(xShareRnd *big.Int) {
	pi.xShareRnd = xShareRnd
}

func (pi *ProverInfo) getRhoI() *big.Int {
	return pi.rhoI
}

func (pi *ProverInfo) setRhoI(rhoI *big.Int) {
	pi.rhoI = rhoI
}

func (pi *ProverInfo) getRhoIRnd() *big.Int {
	return pi.rhoIRnd
}

func (pi *ProverInfo) setRhoIRnd(rhoIRnd *big.Int) {
	pi.rhoIRnd = rhoIRnd
}

func (pi *ProverInfo) getOpenUiVi() *Open {
	return pi.openUiVi
}

func (pi *ProverInfo) setOpenUiVi(openUiVi *Open) {
	pi.openUiVi = openUiVi
}

func (pi *ProverInfo) getOpenRiWi() *Open {
	return pi.openRiWi
}

func (pi *ProverInfo) setOpenRiWi(openRiWi *Open) {
	pi.openRiWi = openRiWi
}

func (pi *ProverInfo) getkI() *big.Int {
	return pi.kI
}

func (pi *ProverInfo) setkI(kI *big.Int) {
	pi.kI = kI
}

func (pi *ProverInfo) getcI() *big.Int {
	return pi.cI
}

func (pi *ProverInfo) setcI(cI *big.Int) {
	pi.cI = cI
}

func (pi *ProverInfo) getcIRnd() *big.Int {
	return pi.cIRnd
}

func (pi *ProverInfo) setcIRnd(cIRnd *big.Int) {
	pi.cIRnd = cIRnd
}

func (pi *ProverInfo) getuI() *big.Int {
	return pi.uI
}

func (pi *ProverInfo) setuI(uI *big.Int) {
	pi.uI = uI
}

func (pi *ProverInfo) getvI() *big.Int {
	return pi.vI
}

func (pi *ProverInfo) setvI(vI *big.Int) {
	pi.vI = vI
}

func (pi *ProverInfo) getwI() *big.Int {
	return pi.wI
}

func (pi *ProverInfo) setwI(wI *big.Int) {
	pi.wI = wI
}

func (pi *ProverInfo) getEncXShare() *big.Int {
	return pi.encXShare
}

func (pi *ProverInfo) setEncXShare(encXShare *big.Int) {
	pi.encXShare = encXShare //加密的私钥片
}

func (pi *ProverInfo) getyShare_x() *big.Int {
	return pi.yShare_x
}

func (pi *ProverInfo) getyShare_y() *big.Int {
	return pi.yShare_y
}

func (pi *ProverInfo) setyShare_x(yShare_x *big.Int) {
	pi.yShare_x = yShare_x
}

func (pi *ProverInfo) setyShare_y(yShare_y *big.Int) {
	pi.yShare_y = yShare_y
}

func (pi *ProverInfo) getMpkUiVi() *MultiTrapdoorCommitment {
	return pi.mpkUiVi
}

func (pi *ProverInfo) setMpkUiVi(mpkUiVi *MultiTrapdoorCommitment) {
	pi.mpkUiVi = mpkUiVi
}

func (pi *ProverInfo) getCmtUiVi() *Commitment {
	return pi.cmtUiVi
}

func (pi *ProverInfo) setCmtUiVi(cmtUiVi *Commitment) {
	pi.cmtUiVi = cmtUiVi
}

func (pi *ProverInfo) getZkp1() *Zkpi1 {
	return pi.zkp1
}

func (pi *ProverInfo) setZkp1(zkp1 *Zkpi1) {
	pi.zkp1 = zkp1
}

func (pi *ProverInfo) getrI_x() *big.Int {
	return pi.rI_x
}

func (pi *ProverInfo) getrI_y() *big.Int {
	return pi.rI_y
}

func (pi *ProverInfo) setrI_x(rI_x *big.Int) {
	pi.rI_x = rI_x
}

func (pi *ProverInfo) setrI_y(rI_y *big.Int) {
	pi.rI_y = rI_y
}

func (pi *ProverInfo) getMask() *big.Int {
	return pi.mask
}

func (pi *ProverInfo) setMask(mask *big.Int) {
	pi.mask = mask
}

func (pi *ProverInfo) getMpkRiWi() *MultiTrapdoorCommitment {
	return pi.mpkRiWi
}

func (pi *ProverInfo) setMpkRiWi(mpkRiWi *MultiTrapdoorCommitment) {
	pi.mpkRiWi = mpkRiWi
}

func (pi *ProverInfo) getCmtRiWi() *Commitment {
	return pi.cmtRiWi
}

func (pi *ProverInfo) setCmtRiWi(cmtRiWi *Commitment) {
	pi.cmtRiWi = cmtRiWi
}

func (pi *ProverInfo) getZkp_i2() *Zkpi2 {
	return pi.zkp_i2
}

func (pi *ProverInfo) setZkp_i2(zkp_i2 *Zkpi2) {
	pi.zkp_i2 = zkp_i2
}

func (pi *ProverInfo) getMpkEncXiYi() *MultiTrapdoorCommitment {
	return pi.mpkEncXiYi
}

func (pi *ProverInfo) setMpkEncXiYi(mpkEncXiYi *MultiTrapdoorCommitment) {
	pi.mpkEncXiYi = mpkEncXiYi
}

func (pi *ProverInfo) getOpenEncXiYi() *Open {
	return pi.openEncXiYi
}

func (pi *ProverInfo) setOpenEncXiYi(openEncXiYi *Open) {
	pi.openEncXiYi = openEncXiYi
}

func (pi *ProverInfo) getCmtEncXiYi() *Commitment {
	return pi.cmtEncXiYi
}

func (pi *ProverInfo) setCmtEncXiYi(cmtEncXiYi *Commitment) {
	pi.cmtEncXiYi = cmtEncXiYi
}

func (pi *ProverInfo) getZkpKG() *Zkp {
	return pi.zkpKG
}

func (pi *ProverInfo) setZkpKG(zkpKG *Zkp) {
	pi.zkpKG = zkpKG
}

func (pi *ProverInfo) GetEncX() *big.Int {
	return pi.encX
}

func (pi *ProverInfo) setEncX(encX *big.Int) {
	pi.encX = encX
}

func (pi *ProverInfo) GetPk_x() *big.Int {
	return pi.pk_x
}

func (pi *ProverInfo) setPk_x(pk_x *big.Int) {
	pi.pk_x = pk_x
}

func (pi *ProverInfo) GetPk_y() *big.Int {
	return pi.pk_y
}

func (pi *ProverInfo) setPk_y(pk_y *big.Int) {
	pi.pk_y = pk_y
}
