package rpc

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	. "github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"
	"github.com/elastos/Elastos.ELA.Arbiter/config"
	"github.com/elastos/Elastos.ELA.Arbiter/log"
	"github.com/elastos/Elastos.ELA.Utility/common"
	elaCore "github.com/elastos/Elastos.ELA/core"
)

type Response struct {
	ID      int64       `json:"id"`
	Version string      `json:"jsonrpc"`
	Result  interface{} `json:"result"`
	*Error  `json:"error"`
}

type Error struct {
	ID      int64  `json:"id"`
	Code    int64  `json:"code"`
	Message string `json:"message"`
}

type ArbitratorGroupInfo struct {
	OnDutyArbitratorIndex int
	Arbitrators           []string
}

type BlockTransactions struct {
	Hash         string
	Height       uint32
	Transactions []*TransactionInfo
}

func GetArbitratorGroupInfoByHeight(height uint32) (*ArbitratorGroupInfo, error) {
	resp, err := CallAndUnmarshal("getarbitratorgroupbyheight", Param("height", height), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}
	groupInfo := &ArbitratorGroupInfo{}
	Unmarshal(&resp, groupInfo)

	return groupInfo, nil
}

func GetCurrentHeight(config *config.RpcConfig) (uint32, error) {
	result, err := CallAndUnmarshal("getblockcount", nil, config)
	if err != nil {
		return 0, err
	}
	return uint32(result.(float64)) - 1, nil
}

func GetBlockByHeight(height uint32, config *config.RpcConfig) (*BlockInfo, error) {
	resp, err := CallAndUnmarshal("getblockbyheight", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	Unmarshal(&resp, block)

	return block, nil
}

func GetBlockByHash(hash *common.Uint256, config *config.RpcConfig) (*BlockInfo, error) {
	hashBytes, err := common.HexStringToBytes(hash.String())
	if err != nil {
		return nil, err
	}
	reversedHashBytes := common.BytesReverse(hashBytes)
	reversedHashStr := common.BytesToHexString(reversedHashBytes)

	resp, err := CallAndUnmarshal("getblock",
		Param("blockhash", reversedHashStr).Add("verbosity", 2), config)
	if err != nil {
		return nil, err
	}
	block := &BlockInfo{}
	Unmarshal(&resp, block)

	return block, nil
}

func GetDestroyedTransactionByHeight(height uint32, config *config.RpcConfig) (*BlockTransactions, error) {
	resp, err := CallAndUnmarshal("getdestroyedtransactions", Param("height", height), config)
	if err != nil {
		return nil, err
	}
	transactions, err := GetBlockTransactions(resp)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

func IsTransactionExist(transactionHash string, config *config.RpcConfig) (bool, error) {
	_, err := CallAndUnmarshal("getrawtransaction", Param("txid", transactionHash), config)
	if err != nil {
		return false, err
	}

	return true, nil
}

func GetTransactionByHash(transactionHash string, config *config.RpcConfig) ([]byte, error) {
	hashBytes, err := common.HexStringToBytes(transactionHash)
	if err != nil {
		return nil, err
	}
	reversedHashBytes := common.BytesReverse(hashBytes)
	reversedHashStr := common.BytesToHexString(reversedHashBytes)

	result, err := CallAndUnmarshal("getrawtransaction", Param("txid", reversedHashStr), config)
	if err != nil {
		return nil, err
	}

	var tx string
	Unmarshal(&result, &tx)

	txBytes, err := common.HexStringToBytes(tx)
	if err != nil {
		return nil, err
	}

	return txBytes, nil
}

func GetExistWithdrawTransactions(txs []string) ([]string, error) {
	infoBytes, err := json.Marshal(txs)
	if err != nil {
		return nil, err
	}

	result, err := CallAndUnmarshal("getexistwithdrawtransactions",
		Param("txs", common.BytesToHexString(infoBytes)), config.Parameters.MainNode.Rpc)
	if err != nil {
		return nil, err
	}

	var removeTxs []string
	Unmarshal(&result, &removeTxs)
	return removeTxs, nil
}

func GetExistDepositTransactions(txs []string, config *config.RpcConfig) ([]string, error) {
	infoBytes, err := json.Marshal(txs)
	if err != nil {
		return nil, err
	}

	result, err := CallAndUnmarshal("getexistdeposittransactions",
		Param("txs", common.BytesToHexString(infoBytes)), config)
	if err != nil {
		return nil, err
	}

	var removeTxs []string
	Unmarshal(&result, &removeTxs)
	return removeTxs, nil
}

func GetUnspendUtxo(addresses []string, config *config.RpcConfig) ([]UTXOInfo, error) {
	parameter := make(map[string][]string)
	parameter["addresses"] = addresses
	result, err := CallAndUnmarshals("listunspent", parameter, config)
	if err != nil {
		return nil, err
	}

	var utxoInfos []UTXOInfo
	Unmarshal(&result, &utxoInfos)

	return utxoInfos, nil
}

func Call(method string, params map[string]string, config *config.RpcConfig) ([]byte, error) {
	url := "http://" + config.IpAddress + ":" + strconv.Itoa(config.HttpJsonPort)
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Info("POST requset err:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func Calls(method string, params map[string][]string, config *config.RpcConfig) ([]byte, error) {
	url := "http://" + config.IpAddress + ":" + strconv.Itoa(config.HttpJsonPort)
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"params": params,
	})
	if err != nil {
		return nil, err
	}

	resp, err := http.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		log.Info("POST requset err:", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func GetBlockTransactions(resp interface{}) (*BlockTransactions, error) {
	transactions := &BlockTransactions{}
	Unmarshal(&resp, transactions)

	for _, txInfo := range transactions.Transactions {
		var assetInfo PayloadInfo
		switch txInfo.TxType {
		case elaCore.RegisterAsset:
			assetInfo = &RegisterAssetInfo{}
		case elaCore.CoinBase:
			assetInfo = &CoinbaseInfo{}
		case elaCore.TransferAsset:
			assetInfo = &TransferAssetInfo{}
		case elaCore.RechargeToSideChain:
			assetInfo = &RechargeToSideChainInfo{}
		case elaCore.TransferCrossChainAsset:
			assetInfo = &TransferCrossChainAssetInfo{}
		default:
			return nil, errors.New("GetBlockTransactions: Unknown payload type")
		}
		err := Unmarshal(&txInfo.Payload, assetInfo)
		if err == nil {
			txInfo.Payload = assetInfo
		}
	}
	return transactions, nil
}

func CallAndUnmarshal(method string, params map[string]string, config *config.RpcConfig) (interface{}, error) {
	body, err := Call(method, params, config)
	if err != nil {
		return nil, err
	}

	resp := Response{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), nil
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Message)
	}

	return resp.Result, nil
}

func CallAndUnmarshals(method string, params map[string][]string, config *config.RpcConfig) (interface{}, error) {
	body, err := Calls(method, params, config)
	if err != nil {
		return nil, err
	}

	resp := Response{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return string(body), nil
	}

	if resp.Error != nil {
		return nil, errors.New(resp.Error.Message)
	}

	return resp.Result, nil
}

func CallAndUnmarshalResponse(method string, params map[string]string, config *config.RpcConfig) (Response, error) {
	body, err := Call(method, params, config)
	if err != nil {
		return Response{}, err
	}

	resp := Response{}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return Response{}, err
	}

	return resp, nil
}

func Unmarshal(result interface{}, target interface{}) error {
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
