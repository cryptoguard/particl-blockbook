package common

import (
	"encoding/json"
	"os"

	"github.com/juju/errors"
)

// Config struct
type Config struct {
	CoinName                string `json:"coin_name"`
	CoinShortcut            string `json:"coin_shortcut"`
	CoinLabel               string `json:"coin_label"`
	Network                 string `json:"network"`
	FourByteSignatures      string `json:"fourByteSignatures"`
	FiatRates               string `json:"fiat_rates"`
	FiatRatesParams         string `json:"fiat_rates_params"`
	FiatRatesVsCurrencies   string `json:"fiat_rates_vs_currencies"`
	BlockGolombFilterP      uint8  `json:"block_golomb_filter_p"`
	BlockFilterScripts      string `json:"block_filter_scripts"`
	BlockFilterUseZeroedKey bool   `json:"block_filter_use_zeroed_key"`
}

// configFile represents the nested JSON structure used in configs/coins/*.json
type configFile struct {
	Coin struct {
		Name     string `json:"name"`
		Shortcut string `json:"shortcut"`
		Label    string `json:"label"`
	} `json:"coin"`
	IPC struct {
		RPCURLTemplate          string `json:"rpc_url_template"`
		RPCUser                 string `json:"rpc_user"`
		RPCPass                 string `json:"rpc_pass"`
		RPCTimeout              int    `json:"rpc_timeout"`
		MessageQueueBindingTemplate string `json:"message_queue_binding_template"`
	} `json:"ipc"`
	Ports struct {
		BackendRPC          int `json:"backend_rpc"`
		BackendMessageQueue int `json:"backend_message_queue"`
		BlockbookInternal   int `json:"blockbook_internal"`
		BlockbookPublic     int `json:"blockbook_public"`
	} `json:"ports"`
	Blockbook struct {
		BlockChain struct {
			Parse            bool   `json:"parse"`
			MempoolWorkers   int    `json:"mempool_workers"`
			MempoolSubWorkers int   `json:"mempool_sub_workers"`
			BlockAddressesToKeep int `json:"block_addresses_to_keep"`
			XPubMagic           uint32 `json:"xpub_magic"`
			XPubMagicSegwitP2sh uint32 `json:"xpub_magic_segwit_p2sh"`
			XPubMagicSegwitNative uint32 `json:"xpub_magic_segwit_native"`
			Slip44              uint32 `json:"slip44"`
			AdditionalParams struct {
				FourByteSignatures      string `json:"fourByteSignatures"`
				FiatRates               string `json:"fiat_rates"`
				FiatRatesParams         string `json:"fiat_rates_params"`
				FiatRatesVsCurrencies   string `json:"fiat_rates_vs_currencies"`
				BlockGolombFilterP      uint8  `json:"block_golomb_filter_p"`
				BlockFilterScripts      string `json:"block_filter_scripts"`
				BlockFilterUseZeroedKey bool   `json:"block_filter_use_zeroed_key"`
			} `json:"additional_params"`
		} `json:"block_chain"`
	} `json:"blockbook"`
}

// UnmarshalJSON implements custom JSON unmarshaling to handle the nested config file format
func (c *Config) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as the flat format (for backward compatibility)
	type Alias Config
	alias := &struct {
		*Alias
	}{
		Alias: (*Alias)(c),
	}
	if err := json.Unmarshal(data, alias); err == nil && c.CoinName != "" {
		// Successfully unmarshaled flat format
		return nil
	}

	// If flat format didn't work or CoinName is empty, try nested format
	var cf configFile
	if err := json.Unmarshal(data, &cf); err != nil {
		return err
	}

	// Map nested structure to flat Config struct
	c.CoinName = cf.Coin.Name
	c.CoinShortcut = cf.Coin.Shortcut
	c.CoinLabel = cf.Coin.Label
	c.FourByteSignatures = cf.Blockbook.BlockChain.AdditionalParams.FourByteSignatures
	c.FiatRates = cf.Blockbook.BlockChain.AdditionalParams.FiatRates
	c.FiatRatesParams = cf.Blockbook.BlockChain.AdditionalParams.FiatRatesParams
	c.FiatRatesVsCurrencies = cf.Blockbook.BlockChain.AdditionalParams.FiatRatesVsCurrencies
	c.BlockGolombFilterP = cf.Blockbook.BlockChain.AdditionalParams.BlockGolombFilterP
	c.BlockFilterScripts = cf.Blockbook.BlockChain.AdditionalParams.BlockFilterScripts
	c.BlockFilterUseZeroedKey = cf.Blockbook.BlockChain.AdditionalParams.BlockFilterUseZeroedKey

	return nil
}

// GetConfig loads and parses the config file and returns Config struct
func GetConfig(configFile string) (*Config, error) {
	if configFile == "" {
		return nil, errors.New("Missing blockchaincfg configuration parameter")
	}

	configFileContent, err := os.ReadFile(configFile)
	if err != nil {
		return nil, errors.Errorf("Error reading file %v, %v", configFile, err)
	}

	var cn Config
	err = json.Unmarshal(configFileContent, &cn)
	if err != nil {
		return nil, errors.Annotatef(err, "Error parsing config file ")
	}
	return &cn, nil
}
