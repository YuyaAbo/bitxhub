package tester

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strconv"

	appchainMgr "github.com/meshplus/bitxhub-core/appchain-mgr"
	"github.com/meshplus/bitxhub-core/governance"
	"github.com/meshplus/bitxhub-core/validator"
	"github.com/meshplus/bitxhub-kit/crypto"
	"github.com/meshplus/bitxhub-kit/crypto/asym"
	"github.com/meshplus/bitxhub-model/constant"
	"github.com/meshplus/bitxhub-model/pb"
	"github.com/meshplus/bitxhub/internal/coreapi/api"
	"github.com/meshplus/bitxhub/internal/executor/contracts"
	"github.com/stretchr/testify/suite"
)

type Governance struct {
	suite.Suite
	api     api.CoreAPI
	privKey crypto.PrivateKey
}

func (suite *Governance) SetupSuite() {
}

func (suite *Governance) TestGovernance() {
	path1 := "./test_data/config/node1/key.json"
	path2 := "./test_data/config/node2/key.json"
	path3 := "./test_data/config/node3/key.json"
	path4 := "./test_data/config/node4/key.json"
	keyPath1 := filepath.Join(path1)
	keyPath2 := filepath.Join(path2)
	keyPath3 := filepath.Join(path3)
	keyPath4 := filepath.Join(path4)
	priAdmin1, err := asym.RestorePrivateKey(keyPath1, "bitxhub")
	suite.Require().Nil(err)
	priAdmin2, err := asym.RestorePrivateKey(keyPath2, "bitxhub")
	suite.Require().Nil(err)
	priAdmin3, err := asym.RestorePrivateKey(keyPath3, "bitxhub")
	suite.Require().Nil(err)
	priAdmin4, err := asym.RestorePrivateKey(keyPath4, "bitxhub")
	suite.Require().Nil(err)
	fromAdmin1, err := priAdmin1.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin2, err := priAdmin2.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin3, err := priAdmin3.PublicKey().Address()
	suite.Require().Nil(err)
	fromAdmin4, err := priAdmin4.PublicKey().Address()
	suite.Require().Nil(err)
	adminNonce2 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin2.String())
	adminNonce3 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin3.String())
	adminNonce4 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin4.String())

	appchainPri, err := asym.GenerateKeyPair(crypto.Secp256k1)
	suite.Require().Nil(err)
	appchainPub, err := appchainPri.PublicKey().Bytes()
	suite.Require().Nil(err)
	addr, err := appchainPri.PublicKey().Address()
	suite.Require().Nil(err)
	appchainNonce := uint64(0)

	suite.Require().Nil(transfer(suite.Suite, suite.api, addr, 10000000000000))
	adminNonce1 := suite.api.Broker().GetPendingNonceByAccount(fromAdmin1.String())

	// 1. Register ==============================================
	ret, err := invokeBVMContract(suite.api, appchainPri, appchainNonce, constant.AppchainMgrContractAddr.Address(), "RegisterV2",
		pb.String(fmt.Sprintf("appchain%s", addr.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String("validators"),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(base64.StdEncoding.EncodeToString(appchainPub)),
		pb.String("reason"),
		pb.String(validator.HappyRuleAddr),
		pb.String(""),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	appchainNonce++
	gRet := &governance.GovernanceResult{}
	err = json.Unmarshal(ret.Ret, gRet)
	suite.Require().Nil(err)
	chainId := string(gRet.Extra)
	registerProposalId := gRet.ProposalID

	// repeated registration
	ret, err = invokeBVMContract(suite.api, appchainPri, appchainNonce, constant.AppchainMgrContractAddr.Address(), "RegisterV2",
		pb.String(fmt.Sprintf("appchain%s", addr.String())),
		pb.String(docAddr),
		pb.String(docHash),
		pb.String("validators"),
		pb.String("rbft"),
		pb.String("hyperchain"),
		pb.String("税务链"),
		pb.String("趣链税务链"),
		pb.String("1.8"),
		pb.String(string(appchainPub)),
		pb.String("reason"),
		pb.String(validator.HappyRuleAddr),
		pb.String(""),
	)
	suite.Require().Nil(err)
	suite.Require().False(ret.IsSuccess(), string(ret.Ret))
	appchainNonce++

	// get proposal
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "GetProposal", pb.String(registerProposalId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	p := contracts.Proposal{}
	err = json.Unmarshal(ret.Ret, &p)
	suite.Require().Nil(err)
	suite.Require().Equal("register", string(p.EventType), "event type")

	// get chain status
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	chain := appchainMgr.Appchain{}
	err = json.Unmarshal(ret.Ret, &chain)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceRegisting, chain.Status)

	// get role weight
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.RoleContractAddr.Address(), "GetRoleWeight", pb.String(fromAdmin1.Address))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	w, err := strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().Equal(2, w, "weight")

	// vote1: approve
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(registerProposalId),
		pb.String(contracts.BallotApprove),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++

	// get ballot
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "GetBallot",
		pb.String(fromAdmin1.Address),
		pb.String(registerProposalId),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	b := &contracts.Ballot{}
	err = json.Unmarshal(ret.Ret, &b)
	suite.Require().Nil(err)
	suite.Require().Equal(string(contracts.APPROVED), b.Approve)

	// vote2: approve
	ret, err = invokeBVMContract(suite.api, priAdmin2, adminNonce2, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(registerProposalId),
		pb.String(contracts.BallotApprove),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce2++

	// vote3: approve -> proposal approve
	ret, err = invokeBVMContract(suite.api, priAdmin3, adminNonce3, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(registerProposalId),
		pb.String(contracts.BallotApprove),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce3++

	// vote4: error, the proposal is closed
	ret, err = invokeBVMContract(suite.api, priAdmin4, adminNonce4, constant.GovernanceContractAddr.Address(), "Vote",
		pb.String(registerProposalId),
		pb.String(contracts.BallotApprove),
		pb.String("reason"),
	)
	suite.Require().Nil(err)
	suite.Require().False(ret.IsSuccess(), string(ret.Ret))
	adminNonce4++

	// get approve num
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "GetApproveNum", pb.String(registerProposalId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	num, err := strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().Equal(3, num, "approveNum")

	// get against num
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "GetAgainstNum", pb.String(registerProposalId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	num, err = strconv.Atoi(string(ret.Ret))
	suite.Require().Nil(err)
	suite.Require().Equal(0, num, "againstNum")

	// get proposal status
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.GovernanceContractAddr.Address(), "GetStatus", pb.String(registerProposalId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	proposalStatus := string(ret.Ret)
	suite.Require().Equal(contracts.APPROVED, contracts.ProposalStatus(proposalStatus))

	// get chain status
	ret, err = invokeBVMContract(suite.api, priAdmin1, adminNonce1, constant.AppchainMgrContractAddr.Address(), "GetAppchain", pb.String(chainId))
	suite.Require().Nil(err)
	suite.Require().True(ret.IsSuccess(), string(ret.Ret))
	adminNonce1++
	err = json.Unmarshal(ret.Ret, &chain)
	suite.Require().Nil(err)
	suite.Require().Equal(governance.GovernanceAvailable, chain.Status)
}
