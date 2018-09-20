package sideauxpow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/arbitrator"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Arbiter/password"
	"github.com/elastos/Elastos.ELA.Arbiter/rpc"
	"github.com/elastos/Elastos.ELA.Arbiter/wallet"

	"github.com/elastos/Elastos.ELA.SideChain/common"
	. "github.com/elastos/Elastos.ELA.Utility/common"
	"github.com/elastos/Elastos.ELA.Utility/crypto"
	ela "github.com/elastos/Elastos.ELA/core"
)

var (
	CurrentWallet       wallet.Wallet
	mainAccountPassword []byte

	LastSendSideMiningHeightMap   map[Uint256]uint32
	LastNotifySideMiningHeightMap map[Uint256]uint32
	LastSubmitAuxpowHeightMap     map[Uint256]uint32
)

func getMainAccountPassword() []byte {
	return mainAccountPassword
}

func SetMainAccountPassword(passwd []byte) {
	mainAccountPassword = passwd
}

func UpdateLastNotifySideMiningHeight(genesisBlockHash Uint256) {
	LastNotifySideMiningHeightMap[genesisBlockHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
}

func UpdateLastSubmitAuxpowHeight(genesisBlockHash Uint256) {
	LastSubmitAuxpowHeightMap[genesisBlockHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()
}

func getPassword(passwd []byte, confirmed bool) []byte {
	var tmp []byte
	var err error
	if len(passwd) > 0 {
		tmp = []byte(passwd)
	} else {
		if confirmed {
			tmp, err = password.GetConfirmedPassword()
		} else {
			tmp, err = password.GetPassword()
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
	return tmp
}

func unmarshal(result interface{}, target interface{}) error {
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	if err != nil {
		return err
	}
	return nil
}

func sideChainPowTransfer(name string, passwd []byte, sideNode *config.SideNodeConfig) error {
	log.Info("getSideAuxpow")
	depositAddress := sideNode.PayToAddr
	if depositAddress == "" {
		return errors.New("Has no side aux pow paytoaddr")
	}
	resp, err := rpc.CallAndUnmarshal("createauxblock", rpc.Param("paytoaddress", depositAddress), sideNode.Rpc)
	if err != nil {
		return err
	}
	if resp == nil {
		log.Info("Create auxblock, nil ")
		return nil
	}

	type SideAuxBlock struct {
		GenesisHash       string `json:"genesishash"`
		Height            uint32 `json:"height"`
		Bits              string `json:"bits"`
		Hash              string `json:"hash"`
		PreviousBlockHash string `json:"previousblockhash"`
	}

	sideAuxBlock := &SideAuxBlock{}

	err = unmarshal(resp, sideAuxBlock)
	if err != nil {
		return err
	}
	// fmt.Println(sideAuxBlock)

	txType := ela.SideChainPow

	sideGenesisHashData, _ := HexStringToBytes(sideAuxBlock.GenesisHash)
	sideBlockHashData, _ := HexStringToBytes(sideAuxBlock.Hash)

	sideGenesisHash, _ := Uint256FromBytes(sideGenesisHashData)
	sideBlockHash, _ := Uint256FromBytes(sideBlockHashData)

	fmt.Println(sideGenesisHash, sideBlockHash)
	// Create payload
	txPayload := &ela.PayloadSideChainPow{
		BlockHeight:     sideAuxBlock.Height,
		SideBlockHash:   *sideBlockHash,
		SideGenesisHash: *sideGenesisHash,
	}

	buf := new(bytes.Buffer)
	txPayload.Serialize(buf, ela.SideChainPowPayloadVersion)
	txPayload.SignedData, err = arbitrator.ArbitratorGroupSingleton.GetCurrentArbitrator().Sign(buf.Bytes()[0:68])
	if err != nil {
		return err
	}

	// create transaction
	if config.Parameters.SideAuxPowFee <= 0 {
		return errors.New("Invalid side aux pow fee")
	}
	fee := Fixed64(config.Parameters.SideAuxPowFee)

	addr := CurrentWallet.GetAddress(name)
	if addr == nil {
		return errors.New("Get key store address failed:" + name)
	}

	from := addr.Addr.Address
	script := addr.Addr.RedeemScript

	var txn *ela.Transaction
	txn, err = CurrentWallet.CreateAuxpowTransaction(txType, txPayload, from, &fee, script, *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight())
	if err != nil {
		return errors.New("create transaction failed: " + err.Error())
	}

	// sign transaction
	program := txn.Programs[0]

	haveSign, needSign, err := crypto.GetSignStatus(program.Code, program.Parameter)
	if haveSign == needSign {
		return errors.New("transaction was fully signed, no need more sign")
	}
	_, err = CurrentWallet.Sign(name, getPassword(passwd, false), txn)
	if err != nil {
		return err
	}
	haveSign, needSign, _ = crypto.GetSignStatus(program.Code, program.Parameter)
	log.Debug("Transaction successfully signed: ", haveSign, needSign)

	sideChainPowBuf := new(bytes.Buffer)
	txn.Serialize(sideChainPowBuf)
	content := BytesToHexString(sideChainPowBuf.Bytes())
	// log.Debug("Raw Sidemining transaction: ", content)

	// send transaction
	result, err := rpc.CallAndUnmarshal("sendrawtransaction", rpc.Param("data", content), config.Parameters.MainNode.Rpc)
	if err != nil {
		return errors.New("[SendSideChainMining] sendrawtransaction failed: " + err.Error())
	}
	log.Info("[SendSideChainMining] End send Sidemining transaction:  genesis address [", sideNode.GenesisBlockAddress, "], result: ", result)

	LastSendSideMiningHeightMap[*sideGenesisHash] = *arbitrator.ArbitratorGroupSingleton.GetCurrentHeight()

	return nil
}

func StartSideChainMining(sideNode *config.SideNodeConfig) {
	err := sideChainPowTransfer(sideNode.KeystoreFile, getMainAccountPassword(), sideNode)
	if err != nil {
		log.Warn(err)
	}
}

func calculateGenesisAddress(genesisBlockHash string) (string, error) {
	genesisBlockBytes, err := HexStringToBytes(genesisBlockHash)
	if err != nil {
		return "", errors.New("genesis block hash string to bytes failed")
	}
	genesisHash, err := Uint256FromBytes(genesisBlockBytes)
	if err != nil {
		return "", errors.New("genesis block hash bytes to hash failed")
	}

	genesisAddress, err := common.GetGenesisAddress(*genesisHash)
	if err != nil {
		return "", errors.New("genesis block hash to genesis address failed")
	}

	return genesisAddress, nil
}

func TestMultiSidechain() {
	for {
		select {
		case <-time.After(time.Second * 3):
			for _, node := range config.Parameters.SideNodeList {
				StartSideChainMining(node)
			}
			println("TestMultiSidechain")
		}
	}
}

func init() {
	LastSendSideMiningHeightMap = make(map[Uint256]uint32)
	LastNotifySideMiningHeightMap = make(map[Uint256]uint32)
	LastSubmitAuxpowHeightMap = make(map[Uint256]uint32)
}
