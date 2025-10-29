package part

import (
	"encoding/json"

	"github.com/golang/glog"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
	"github.com/trezor/blockbook/common"
)

// ParticlRPC is an interface to JSON-RPC particld service.
type ParticlRPC struct {
	*btc.BitcoinRPC
}

// NewParticlRPC returns new ParticlRPC instance.
func NewParticlRPC(config json.RawMessage, pushHandler func(bchain.NotificationType)) (bchain.BlockChain, error) {
	b, err := btc.NewBitcoinRPC(config, pushHandler)
	if err != nil {
		return nil, err
	}

	s := &ParticlRPC{
		b.(*btc.BitcoinRPC),
	}
	s.RPCMarshaler = btc.JSONMarshalerV2{}
	s.ChainConfig.SupportsEstimateFee = true
	s.ChainConfig.SupportsEstimateSmartFee = true

	return s, nil
}

// Initialize initializes ParticlRPC instance.
func (b *ParticlRPC) Initialize() error {
	ci, err := b.GetChainInfo()
	if err != nil {
		return err
	}
	chainName := ci.Chain

	glog.Info("Chain name ", chainName)
	params := GetChainParams(chainName)

	// always create parser
	b.Parser = NewParticlParser(params, b.ChainConfig)

	// parameters for getInfo request
	if params.Net == MainnetMagic {
		b.Testnet = false
		b.Network = "livenet"
	} else {
		b.Testnet = true
		b.Network = "testnet"
	}

	glog.Info("rpc: block chain ", params.Name)

	return nil
}

// ResGetBlockChainInfoParticl is response to getblockchaininfo for Particl
// Particl returns warnings as an array (like modern Bitcoin Core), not a string
type ResGetBlockChainInfoParticl struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Chain         string            `json:"chain"`
		Blocks        int               `json:"blocks"`
		Headers       int               `json:"headers"`
		Bestblockhash string            `json:"bestblockhash"`
		Difficulty    common.JSONNumber `json:"difficulty"`
		SizeOnDisk    int64             `json:"size_on_disk"`
		Warnings      []string          `json:"warnings"` // Array instead of string
	} `json:"result"`
}

// GetChainInfo returns information about the connected backend with Particl-specific handling
// ResGetNetworkInfoParticl is response to getnetworkinfo for Particl
// Particl returns subversion as a string, not a number, and warnings as an array
type ResGetNetworkInfoParticl struct {
	Error  *bchain.RPCError `json:"error"`
	Result struct {
		Version         common.JSONNumber `json:"version"`
		Subversion      string            `json:"subversion"` // String for Particl
		ProtocolVersion common.JSONNumber `json:"protocolversion"`
		Timeoffset      float64           `json:"timeoffset"`
		Warnings        []string          `json:"warnings"` // Array for Particl (like getblockchaininfo)
	} `json:"result"`
}

func (b *ParticlRPC) GetChainInfo() (*bchain.ChainInfo, error) {
	glog.V(1).Info("rpc: getblockchaininfo")

	res := ResGetBlockChainInfoParticl{}
	req := btc.CmdGetBlockChainInfo{Method: "getblockchaininfo"}
	err := b.Call(&req, &res)

	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, res.Error
	}

	// Also get network info for version/subversion
	glog.V(2).Info("particl: GetChainInfo calling getnetworkinfo")
	resNi := ResGetNetworkInfoParticl{}
	reqNi := struct {
		Method string `json:"method"`
	}{Method: "getnetworkinfo"}
	errNi := b.Call(&reqNi, &resNi)
	glog.V(2).Infof("particl: GetChainInfo getnetworkinfo returned, errNi=%v, resNi.Error=%v", errNi, resNi.Error)

	// Convert warnings array to single string
	warningsStr := ""
	if len(res.Result.Warnings) > 0 {
		warningsStr = res.Result.Warnings[0]
		for i := 1; i < len(res.Result.Warnings); i++ {
			warningsStr += "; " + res.Result.Warnings[i]
		}
	}

	rv := &bchain.ChainInfo{
		Bestblockhash: res.Result.Bestblockhash,
		Blocks:        res.Result.Blocks,
		Chain:         res.Result.Chain,
		Difficulty:    res.Result.Difficulty.String(),
		Headers:       res.Result.Headers,
		SizeOnDisk:    res.Result.SizeOnDisk,
		Warnings:      warningsStr,
	}

	// Add version/subversion from getnetworkinfo if available
	if errNi == nil && resNi.Error == nil {
		rv.Version = resNi.Result.Version.String()
		rv.Subversion = resNi.Result.Subversion
		rv.ProtocolVersion = resNi.Result.ProtocolVersion.String()
		rv.Timeoffset = resNi.Result.Timeoffset
		glog.V(2).Infof("particl: GetChainInfo extracted version=%s, subversion=%s", rv.Version, rv.Subversion)
	} else {
		if errNi != nil {
			glog.V(2).Infof("particl: GetChainInfo getnetworkinfo RPC error: %v", errNi)
		}
		if resNi.Error != nil {
			glog.V(2).Infof("particl: GetChainInfo getnetworkinfo result error: %v", resNi.Error)
		}
	}

	return rv, nil
}

// GetBlock overrides BitcoinRPC.GetBlock to ensure our GetBlockFull override is called
func (b *ParticlRPC) GetBlock(hash string, height uint32) (*bchain.Block, error) {
	var err error
	if hash == "" {
		hash, err = b.GetBlockHash(height)
		if err != nil {
			return nil, err
		}
	}
	if !b.ParseBlocks {
		// Call OUR GetBlockFull, not the parent's
		return b.GetBlockFull(hash)
	}
	// For binary parsing, use parent implementation
	if height > 0 {
		return b.BitcoinRPC.GetBlockWithoutHeader(hash, height)
	}
	header, err := b.GetBlockHeader(hash)
	if err != nil {
		return nil, err
	}
	data, err := b.GetBlockBytes(hash)
	if err != nil {
		return nil, err
	}
	block, err := b.Parser.ParseBlock(data)
	if err != nil {
		return nil, err
	}
	block.BlockHeader = *header
	return block, nil
}

// ParticlBlockResult is the result structure for getblock RPC with verbosity=2
// This preserves raw JSON for each transaction to extract Particl-specific fields
type ParticlBlockResult struct {
	bchain.BlockHeader
	Txs []json.RawMessage `json:"tx"`
}

// ResGetBlockFullParticl is the response for getblock RPC (verbosity=2) for Particl
type ResGetBlockFullParticl struct {
	Error  *bchain.RPCError    `json:"error"`
	Result *ParticlBlockResult `json:"result"`
}

// GetBlockFull overrides BitcoinRPC.GetBlockFull to populate CoinSpecificData with CT fees
// This ensures privacy transaction fees are extracted and stored during blockchain sync
func (b *ParticlRPC) GetBlockFull(hash string) (*bchain.Block, error) {
	glog.V(1).Info("rpc: getblock (verbosity=2) ", hash)

	// Use custom response structure that preserves raw transaction JSON
	res := ResGetBlockFullParticl{}
	req := btc.CmdGetBlock{Method: "getblock"}
	req.Params.BlockHash = hash
	req.Params.Verbosity = 2
	err := b.Call(&req, &res)

	if err != nil {
		return nil, err
	}
	if res.Error != nil {
		return nil, res.Error
	}

	// Parse each transaction from raw JSON to extract Particl-specific fields
	txs := make([]bchain.Tx, len(res.Result.Txs))
	for i := range res.Result.Txs {
		tx, err := b.Parser.ParseTxFromJson(res.Result.Txs[i])
		if err != nil {
			glog.Warningf("particl: failed to parse tx from JSON: %v", err)
			continue
		}
		// Log CT fee extraction for privacy transaction monitoring
		if tx.CoinSpecificData != nil {
			if particlData, ok := tx.CoinSpecificData.(*ParticlTxData); ok && particlData.CTFee > 0 {
				glog.V(1).Infof("particl: GetBlockFull extracted CT fee for tx %s: %f PART", tx.Txid, particlData.CTFee)
			}
		}
		txs[i] = *tx
	}

	block := &bchain.Block{
		BlockHeader: res.Result.BlockHeader,
		Txs:         txs,
	}

	return block, nil
}
