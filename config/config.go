package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/elastos/Elastos.ELA.Arbiter/arbitration/base"

	"github.com/elastos/Elastos.ELA/common"
)

const (
	DefaultConfigFilename = "./config.json"
)

var (
	Version    string
	Parameters configParams

	DataPath   = "elastos_arbiter"
	DataDir    = "data"
	SpvDir     = "spv"
	LogDir     = "logs"
	ArbiterDir = "arbiter"
)

type CrossChainArbiterInfo struct {
	PublicKey  string `json:"PublicKey"`
	NetAddress string `json:"NetAddress"`
}

type RpcConfiguration struct {
	User        string   `json:"User"`
	Pass        string   `json:"Pass"`
	WhiteIPList []string `json:"WhiteIPList"`
}

type Configuration struct {
	Magic    uint32 `json:"Magic"`
	Version  uint32 `json:"Version"`
	NodePort uint16 `json:"NodePort"`

	MainNode     *MainNodeConfig   `json:"MainNode"`
	SideNodeList []*SideNodeConfig `json:"SideNodeList"`

	SyncInterval  time.Duration `json:"SyncInterval"`
	HttpJsonPort  int           `json:"HttpJsonPort"`
	HttpRestPort  uint16        `json:"HttpRestPort"`
	PrintLevel    uint8         `json:"PrintLevel"`
	SPVPrintLevel uint8         `json:"SPVPrintLevel"`
	MaxLogsSize   int64         `json:"MaxLogsSize"`
	MaxPerLogSize int64         `json:"MaxPerLogSize"`

	SideChainMonitorScanInterval time.Duration           `json:"SideChainMonitorScanInterval"`
	ClearTransactionInterval     time.Duration           `json:"ClearTransactionInterval"`
	MinOutbound                  int                     `json:"MinOutbound"`
	MaxConnections               int                     `json:"MaxConnections"`
	SideAuxPowFee                int                     `json:"SideAuxPowFee"`
	MinThreshold                 int                     `json:"MinThreshold"`
	DepositAmount                int                     `json:"DepositAmount"`
	CRCOnlyDPOSHeight            uint32                  `json:"CRCOnlyDPOSHeight"`
	MaxTxsPerWithdrawTx          int                     `json:"MaxTxsPerWithdrawTx"`
	OriginCrossChainArbiters     []CrossChainArbiterInfo `json:"OriginCrossChainArbiters"`
	CRCCrossChainArbiters        []CrossChainArbiterInfo `json:"CRCCrossChainArbiters"`
	RpcConfiguration             RpcConfiguration        `json:"RpcConfiguration"`
}

type RpcConfig struct {
	IpAddress    string `json:"IpAddress"`
	HttpJsonPort int    `json:"HttpJsonPort"`
	User         string `json:"User"`
	Pass         string `json:"Pass"`
}

type MainNodeConfig struct {
	Rpc               *RpcConfig `json:"Rpc"`
	SpvSeedList       []string   `json:"SpvSeedList""`
	DefaultPort       uint16     `json:"DefaultPort"`
	Magic             uint32     `json:"Magic"`
	MinOutbound       int        `json:"MinOutbound"`
	MaxConnections    int        `json:"MaxConnections"`
	FoundationAddress string     `json:"FoundationAddress"`
}

type SideNodeConfig struct {
	Rpc *RpcConfig `json:"Rpc"`

	ExchangeRate        float64 `json:"ExchangeRate"`
	GenesisBlockAddress string  `json:"GenesisBlockAddress"`
	GenesisBlock        string  `json:"GenesisBlock"`
	KeystoreFile        string  `json:"KeystoreFile"`
	MiningAddr          string  `json:"MiningAddr"`
	PayToAddr           string  `json:"PayToAddr"`
	PowChain            bool    `json:"PowChain"`
}

type ConfigFile struct {
	ConfigFile Configuration `json:"Configuration"`
}

type configParams struct {
	*Configuration
}

func GetRpcConfig(genesisBlockHash string) (*RpcConfig, bool) {
	for _, node := range Parameters.SideNodeList {
		if node.GenesisBlock == genesisBlockHash {
			return node.Rpc, true
		}
	}
	return nil, false
}

func init() {

	config := ConfigFile{
		ConfigFile: Configuration{
			Magic:                        0,
			Version:                      0,
			NodePort:                     20538,
			HttpJsonPort:                 20536,
			HttpRestPort:                 20534,
			PrintLevel:                   1,
			SPVPrintLevel:                1,
			SyncInterval:                 1000,
			SideChainMonitorScanInterval: 1000,
			ClearTransactionInterval:     60000,
			MinOutbound:                  3,
			MaxConnections:               8,
			SideAuxPowFee:                50000,
			MinThreshold:                 10000000,
			DepositAmount:                10000000,
			MaxTxsPerWithdrawTx:          1000,
			RpcConfiguration: RpcConfiguration{
				User:        "",
				Pass:        "",
				WhiteIPList: []string{"127.0.0.1"},
			},
		},
	}
	Parameters.Configuration = &(config.ConfigFile)

	file, e := ioutil.ReadFile(DefaultConfigFilename)
	if e != nil {
		fmt.Printf("File error: %v\n", e)
		return
	}

	// Remove the UTF-8 Byte Order Mark
	file = bytes.TrimPrefix(file, []byte("\xef\xbb\xbf"))

	e = json.Unmarshal(file, &config)
	if e != nil {
		fmt.Printf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}

	for _, side := range config.ConfigFile.SideNodeList {
		side.PowChain = true
	}

	e = json.Unmarshal(file, &config)
	if e != nil {
		fmt.Printf("Unmarshal json file erro %v", e)
		os.Exit(1)
	}

	//Parameters.Configuration = &(config.ConfigFile)

	var out bytes.Buffer
	err := json.Indent(&out, file, "", "")
	if err != nil {
		fmt.Printf("Config file error: %v\n", e)
		os.Exit(1)
	}

	if Parameters.Configuration.MainNode == nil {
		fmt.Printf("Need to set main node in config file\n")
		return
	}

	if Parameters.Configuration.SideNodeList == nil {
		fmt.Printf("Need to set side node list in config file\n")
		return
	}

	for _, node := range Parameters.SideNodeList {
		genesisBytes, err := common.HexStringToBytes(node.GenesisBlock)
		if err != nil {
			fmt.Printf("Side node genesis block hash error: %v\n", e)
			return
		}
		reversedGenesisBytes := common.BytesReverse(genesisBytes)
		reversedGenesisStr := common.BytesToHexString(reversedGenesisBytes)
		genesisBlockHash, err := common.Uint256FromHexString(reversedGenesisStr)
		if err != nil {
			fmt.Printf("Side node genesis block hash reverse error: %v\n", e)
			return
		}
		address, err := base.GetGenesisAddress(*genesisBlockHash)
		if err != nil {
			fmt.Printf("Side node genesis block hash to address error: %v\n", e)
			return
		}
		node.GenesisBlockAddress = address
		node.GenesisBlock = reversedGenesisStr
	}
}
