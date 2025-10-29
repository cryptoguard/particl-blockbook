package part

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/big"

	"github.com/martinboehm/btcd/wire"
	"github.com/martinboehm/btcutil/base58"
	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
	"github.com/trezor/blockbook/common"
)

// magic numbers
const (
	MainnetMagic wire.BitcoinNet = 0xfbf2efb4
	TestnetMagic wire.BitcoinNet = 0x0b110907
	RegtestMagic wire.BitcoinNet = 0xfabfb5da
)

// chain parameters
var (
	MainNetParams chaincfg.Params
	TestNetParams chaincfg.Params
	RegtestParams chaincfg.Params
)

func init() {
	// Particl mainnet Address encoding magics
	MainNetParams = chaincfg.MainNetParams
	MainNetParams.Name = "Particl"
	MainNetParams.Net = MainnetMagic
	MainNetParams.PubKeyHashAddrID = []byte{56}  // starting with 'P'
	MainNetParams.ScriptHashAddrID = []byte{60}  // starting with 'p'
	MainNetParams.PrivateKeyID = []byte{108}
	MainNetParams.Bech32HRPSegwit = "pw"

	// Particl testnet Address encoding magics
	TestNetParams = chaincfg.TestNet3Params
	TestNetParams.Name = "Particl Testnet"
	TestNetParams.Net = TestnetMagic
	TestNetParams.PubKeyHashAddrID = []byte{118} // starting with 'n'
	TestNetParams.ScriptHashAddrID = []byte{122} // starting with 'r'
	TestNetParams.PrivateKeyID = []byte{46}
	TestNetParams.Bech32HRPSegwit = "tp"

	// Particl regtest Address encoding magics
	RegtestParams = chaincfg.RegressionNetParams
	RegtestParams.Name = "Particl Regtest"
	RegtestParams.Net = RegtestMagic
	RegtestParams.PubKeyHashAddrID = []byte{118}
	RegtestParams.ScriptHashAddrID = []byte{122}
	RegtestParams.PrivateKeyID = []byte{46}
	RegtestParams.Bech32HRPSegwit = "tpw"
}

// ParticlParser handle
type ParticlParser struct {
	*btc.BitcoinLikeParser
	baseparser                         *bchain.BaseParser
	BitcoinOutputScriptToAddressesFunc btc.OutputScriptToAddressesFunc
}

// NewParticlParser returns new ParticlParser instance
func NewParticlParser(params *chaincfg.Params, c *btc.Configuration) *ParticlParser {
	p := &ParticlParser{
		BitcoinLikeParser: btc.NewBitcoinLikeParser(params, c),
		baseparser:        &bchain.BaseParser{},
	}
	p.BitcoinOutputScriptToAddressesFunc = p.OutputScriptToAddressesFunc
	p.OutputScriptToAddressesFunc = p.outputScriptToAddresses
	p.VSizeSupport = true
	return p
}

// GetAddrDescFromAddress returns internal address representation from string address
// Handles Particl-specific P2CS and P2SH256 addresses
func (p *ParticlParser) GetAddrDescFromAddress(address string) (bchain.AddressDescriptor, error) {
	// Try standard address parsing first (handles P2PKH, P2SH, bech32, etc.)
	desc, err := p.BitcoinLikeParser.GetAddrDescFromAddress(address)
	if err == nil {
		return desc, nil
	}

	// Not a standard address, check for Particl-specific types
	// Decode without checksum validation first to check version byte
	decoded := base58.Decode(address)
	if len(decoded) < 5 { // Need at least version + some data + 4-byte checksum
		return nil, bchain.ErrAddressMissing
	}

	version := decoded[0]
	payload := decoded[1 : len(decoded)-4]
	checksum := decoded[len(decoded)-4:]

	// Validate checksum using double SHA256
	hash1 := sha256.Sum256(decoded[:len(decoded)-4])
	hash2 := sha256.Sum256(hash1[:])
	if !bytes.Equal(hash2[:4], checksum) {
		return nil, bchain.ErrAddressMissing
	}

	// Check for P2CS (version 0x39) - 32-byte hash
	if version == 0x39 && len(payload) == 32 {
		// P2CS (Pay-to-Cold-Staking) - 32-byte hash
		// Script: OP_DUP OP_HASH256 <32-byte-hash> OP_EQUALVERIFY OP_CHECKSIG
		script := make([]byte, 37)
		script[0] = 0x76 // OP_DUP
		script[1] = 0xa8 // OP_HASH256
		script[2] = 0x20 // Push 32 bytes
		copy(script[3:35], payload)
		script[35] = 0x88 // OP_EQUALVERIFY
		script[36] = 0xac // OP_CHECKSIG
		return bchain.AddressDescriptor(script), nil
	}

	// Check for P2SH256 (version 0x3d) - 32-byte hash
	if version == 0x3d && len(payload) == 32 {
		// P2SH256 - 32-byte script hash
		// Script: OP_HASH256 <32-byte-hash> OP_EQUAL
		script := make([]byte, 35)
		script[0] = 0xa8 // OP_HASH256
		script[1] = 0x20 // Push 32 bytes
		copy(script[2:34], payload)
		script[34] = 0x87 // OP_EQUAL
		return bchain.AddressDescriptor(script), nil
	}

	// Unable to decode address
	return nil, bchain.ErrAddressMissing
}

// GetChainParams contains network parameters for the main Particl network
func GetChainParams(chain string) *chaincfg.Params {
	if !chaincfg.IsRegistered(&MainNetParams) {
		err := chaincfg.Register(&MainNetParams)
		if err == nil {
			err = chaincfg.Register(&TestNetParams)
		}
		if err == nil {
			err = chaincfg.Register(&RegtestParams)
		}
		if err != nil {
			panic(err)
		}
	}
	// Always ensure names are set (chaincfg.Register might not preserve them)
	MainNetParams.Name = "Particl"
	TestNetParams.Name = "Particl Testnet"
	RegtestParams.Name = "Particl Regtest"

	switch chain {
	case "test":
		return &TestNetParams
	case "regtest":
		return &RegtestParams
	default:
		return &MainNetParams
	}
}

// ParseBlock parses raw block to our Block struct
func (p *ParticlParser) ParseBlock(b []byte) (*bchain.Block, error) {
	r := bytes.NewReader(b)
	w := wire.MsgBlock{}
	h := wire.BlockHeader{}

	// Deserialize the standard 80-byte Bitcoin block header
	err := h.Deserialize(r)
	if err != nil {
		return nil, err
	}

	// Particl blocks have an additional 32-byte witness merkle root after the standard header
	// Skip past it to get to the transaction data
	_, err = r.Seek(32, 1) // 1 = io.SeekCurrent
	if err != nil {
		return nil, err
	}

	// Now decode just the transactions (not the whole block)
	txCount, err := wire.ReadVarInt(r, 0)
	if err != nil {
		return nil, err
	}

	w.Transactions = make([]*wire.MsgTx, 0, txCount)
	for i := uint64(0); i < txCount; i++ {
		tx := wire.MsgTx{}
		err := tx.BtcDecode(r, 0, wire.BaseEncoding)
		if err != nil {
			return nil, err
		}
		w.Transactions = append(w.Transactions, &tx)
	}

	txs := make([]bchain.Tx, len(w.Transactions))
	for ti, t := range w.Transactions {
		txs[ti] = p.TxFromMsgTx(t, false)
	}

	return &bchain.Block{
		BlockHeader: bchain.BlockHeader{
			Size: len(b),
			Time: h.Timestamp.Unix(),
		},
		Txs: txs,
	}, nil
}

// PackTx packs transaction to byte array using protobuf
func (p *ParticlParser) PackTx(tx *bchain.Tx, height uint32, blockTime int64) ([]byte, error) {
	return p.baseparser.PackTx(tx, height, blockTime)
}

// UnpackTx unpacks transaction from protobuf byte array
func (p *ParticlParser) UnpackTx(buf []byte) (*bchain.Tx, uint32, error) {
	return p.baseparser.UnpackTx(buf)
}

// ParseTx parses byte array containing transaction and returns Tx struct
func (p *ParticlParser) ParseTx(b []byte) (*bchain.Tx, error) {
	t := wire.MsgTx{}
	r := bytes.NewReader(b)
	if err := t.Deserialize(r); err != nil {
		return nil, err
	}
	tx := p.TxFromMsgTx(&t, true)
	tx.Hex = hex.EncodeToString(b)
	return &tx, nil
}

// TxFromMsgTx converts wire.MsgTx to bchain.Tx
func (p *ParticlParser) TxFromMsgTx(t *wire.MsgTx, parseAddresses bool) bchain.Tx {
	vin := make([]bchain.Vin, len(t.TxIn))
	for i, in := range t.TxIn {
		// Check if this is a coinstake transaction (Particl PoS)
		if len(t.TxIn) > 0 && i == 0 && t.TxIn[0].PreviousOutPoint.Hash.String() == "0000000000000000000000000000000000000000000000000000000000000000" {
			vin[i] = bchain.Vin{
				Coinbase: hex.EncodeToString(in.SignatureScript),
				Sequence: in.Sequence,
			}
			continue
		}

		s := bchain.ScriptSig{
			Hex: hex.EncodeToString(in.SignatureScript),
		}

		txid := in.PreviousOutPoint.Hash.String()

		vin[i] = bchain.Vin{
			Txid:      txid,
			Vout:      in.PreviousOutPoint.Index,
			Sequence:  in.Sequence,
			ScriptSig: s,
		}
	}

	vout := make([]bchain.Vout, len(t.TxOut))
	for i, out := range t.TxOut {
		addrs := []string{}
		if parseAddresses {
			addrs, _, _ = p.OutputScriptToAddressesFunc(out.PkScript)
		}
		s := bchain.ScriptPubKey{
			Hex:       hex.EncodeToString(out.PkScript),
			Addresses: addrs,
		}
		var vs big.Int
		vs.SetInt64(out.Value)
		vout[i] = bchain.Vout{
			ValueSat:     vs,
			N:            uint32(i),
			ScriptPubKey: s,
		}
	}

	tx := bchain.Tx{
		Txid:     t.TxHash().String(),
		Version:  t.Version,
		LockTime: t.LockTime,
		Vin:      vin,
		Vout:     vout,
	}
	return tx
}

// ParticlScriptPubKey extends the standard ScriptPubKey with Particl-specific fields
type ParticlScriptPubKey struct {
	Hex       string   `json:"hex,omitempty"`
	Addresses []string `json:"addresses"`
	Address   string   `json:"address"`   // For cold staking outputs
	Type      string   `json:"type"`      // Script type (e.g., "nonstandard")
}

// ParticlVin extends the standard Vin with Particl-specific fields for anon inputs
type ParticlVin struct {
	Coinbase  string           `json:"coinbase"`
	Txid      string           `json:"txid"`
	Vout      uint32           `json:"vout"`
	ScriptSig bchain.ScriptSig `json:"scriptSig"`
	Sequence  uint32           `json:"sequence"`

	// Particl anon (RingCT) input fields
	InputType   string   `json:"type,omitempty"`       // "anon" for RingCT inputs
	AnonInputs  uint32   `json:"num_inputs,omitempty"` // Number of real inputs (e.g., 1)
	RingSize    uint32   `json:"ring_size,omitempty"`  // Size of the ring/decoy set (e.g., 5)
}

// ParticlVout extends the standard Vout to use ParticlScriptPubKey
type ParticlVout struct {
	ValueSat        big.Int
	JsonValue       common.JSONNumber   `json:"value"`
	N               uint32              `json:"n"`
	ScriptPubKey    ParticlScriptPubKey `json:"scriptPubKey"`
	OutputType      string              `json:"type,omitempty"`
	ValueCommitment string              `json:"valueCommitment,omitempty"`
	Data            string              `json:"data,omitempty"`
	RangeProof      string              `json:"rangeproof,omitempty"`
	CTFee           float64             `json:"ct_fee,omitempty"` // CT fee in PART for anon/blind txs
}

// ParticlTx is used for unmarshaling Particl transaction JSON
type ParticlTx struct {
	Hex           string        `json:"hex"`
	Txid          string        `json:"txid"`
	Version       int32         `json:"version"`
	LockTime      uint32        `json:"locktime"`
	Vin           []ParticlVin  `json:"vin"`
	Vout          []ParticlVout `json:"vout"`
	Confirmations uint32        `json:"confirmations"`
	Blocktime     int64         `json:"blocktime"`
	Time          int64         `json:"time"`
}

// ParticlTxData contains Particl-specific transaction data
type ParticlTxData struct {
	CTFee float64 // CT fee in PART for blind/anon transactions
}

// ParseTxFromJson parses JSON message containing transaction and returns Tx struct
func (p *ParticlParser) ParseTxFromJson(msg json.RawMessage) (*bchain.Tx, error) {
	var ptx ParticlTx
	err := json.Unmarshal(msg, &ptx)
	if err != nil {
		return nil, err
	}

	// Convert ParticlVin to bchain.Vin
	vin := make([]bchain.Vin, len(ptx.Vin))
	for i := range ptx.Vin {
		pvin := &ptx.Vin[i]

		vin[i] = bchain.Vin{
			Coinbase:  pvin.Coinbase,
			Txid:      pvin.Txid,
			Vout:      pvin.Vout,
			ScriptSig: pvin.ScriptSig,
			Sequence:  pvin.Sequence,

			// Particl anon input fields
			InputType:   pvin.InputType,
			AnonInputs:  pvin.AnonInputs,
			RingSize:    pvin.RingSize,
		}
	}

	// Convert ParticlVout to bchain.Vout
	vout := make([]bchain.Vout, len(ptx.Vout))
	for i := range ptx.Vout {
		pvout := &ptx.Vout[i]

		// Handle Particl privacy outputs (blind/anon)
		// For blind and anon outputs, the "value" field is absent or 0
		// The actual value is encrypted in the commitment
		if pvout.OutputType == "blind" || pvout.OutputType == "anon" {
			// Mark value as unknown (0) for privacy outputs
			vout[i].ValueSat.SetInt64(0)
			vout[i].JsonValue = ""
		} else if pvout.OutputType == "data" {
			// Data outputs have no value
			vout[i].ValueSat.SetInt64(0)
			vout[i].JsonValue = ""
		} else {
			// Standard outputs: convert vout.JsonValue to big.Int and clear it
			vout[i].ValueSat, err = p.AmountToBigInt(pvout.JsonValue)
			if err != nil {
				return nil, err
			}
			vout[i].JsonValue = ""
		}

		// Handle addresses for different output types
		addresses := pvout.ScriptPubKey.Addresses
		if len(addresses) == 0 && pvout.ScriptPubKey.Address != "" {
			// Cold staking outputs have a single "address" field
			addresses = []string{pvout.ScriptPubKey.Address}
		}
		if addresses == nil {
			addresses = []string{}
		}

		vout[i].N = pvout.N
		vout[i].ScriptPubKey = bchain.ScriptPubKey{
			Hex:       pvout.ScriptPubKey.Hex,
			Addresses: addresses,
		}
		vout[i].OutputType = pvout.OutputType
		vout[i].ValueCommitment = pvout.ValueCommitment
		vout[i].Data = pvout.Data
		vout[i].RangeProof = pvout.RangeProof
	}

	// Extract CT fee from the first data output (for blind/anon transactions)
	var ctFee float64
	for i := range ptx.Vout {
		if ptx.Vout[i].OutputType == "data" && ptx.Vout[i].CTFee > 0 {
			ctFee = ptx.Vout[i].CTFee
			break
		}
	}

	tx := &bchain.Tx{
		Hex:              ptx.Hex,
		Txid:             ptx.Txid,
		Version:          ptx.Version,
		LockTime:         ptx.LockTime,
		Vin:              vin,
		Vout:             vout,
		Confirmations:    ptx.Confirmations,
		Blocktime:        ptx.Blocktime,
		Time:             ptx.Time,
		CoinSpecificData: &ParticlTxData{CTFee: ctFee},
	}

	return tx, nil
}

// outputScriptToAddresses converts ScriptPubKey to addresses
// Handles standard Bitcoin-like transactions (P2PKH, P2SH, P2WPKH, P2WSH, etc.)
// and Particl-specific P2CS (cold staking) transactions
// Note: For CT (blind) and RingCT (anon) transactions in RPC mode,
// addresses are extracted from JSON by ParseTxFromJson, not from raw scripts
func (p *ParticlParser) outputScriptToAddresses(script []byte) ([]string, bool, error) {
	// Check for Particl coinstake scripts (composite script with P2PKH + staking address)
	// Two variants:
	// 1. 66 bytes: OP_ISCOINSTAKE OP_IF <P2PKH 25-bytes> OP_ELSE <P2CS 37-bytes> OP_ENDIF
	// 2. 64 bytes: OP_ISCOINSTAKE OP_IF <P2PKH 25-bytes> OP_ELSE <P2SH256 35-bytes> OP_ENDIF

	if (len(script) == 66 || len(script) == 64) &&
		script[0] == 0xb8 && // OP_ISCOINSTAKE
		script[1] == 0x63 { // OP_IF

		// The P2PKH portion starts at position 2
		// Check for standard P2PKH: OP_DUP OP_HASH160 PUSH20 <20-bytes> OP_EQUALVERIFY OP_CHECKSIG (25 bytes)
		if script[2] == 0x76 && // OP_DUP
			script[3] == 0xa9 && // OP_HASH160
			script[4] == 0x14 { // PUSH20

			// P2PKH is 25 bytes (positions 2-26), so OP_ELSE should be at position 27
			if script[27] == 0x67 && script[len(script)-1] == 0x68 { // OP_ELSE and OP_ENDIF
				// Extract staking script (between ELSE and ENDIF)
				stakingScript := script[28 : len(script)-1]

				// Check for P2CS format: OP_DUP OP_SHA256 <32 bytes> OP_EQUALVERIFY OP_CHECKSIG (37 bytes)
				if len(stakingScript) == 37 &&
					stakingScript[0] == 0x76 && // OP_DUP
					stakingScript[1] == 0xa8 && // OP_SHA256
					stakingScript[2] == 0x20 && // Push 32 bytes
					stakingScript[35] == 0x88 && // OP_EQUALVERIFY
					stakingScript[36] == 0xac { // OP_CHECKSIG

					// P2CS address (version 0x39, starts with "2")
					pubKeyHash := stakingScript[3:35]
					addr := &ParticlP2CSAddress{
						pubKeyHash:  pubKeyHash,
						versionByte: 0x39, // PUBKEY_ADDRESS_256
						params:      p.Params,
					}
					return []string{addr.String()}, true, nil
				}

				// Check for P2SH256 format: OP_SHA256 PUSH32 <32 bytes> OP_EQUAL (35 bytes)
				if len(stakingScript) == 35 &&
					stakingScript[0] == 0xa8 && // OP_SHA256
					stakingScript[1] == 0x20 && // Push 32 bytes
					stakingScript[34] == 0x87 { // OP_EQUAL

					// P2SH256 address (version 0x3d, starts with "33")
					scriptHash := stakingScript[2:34]
					addr := &ParticlP2CSAddress{
						pubKeyHash:  scriptHash,
						versionByte: 0x3d, // SCRIPT_ADDRESS_256
						params:      p.Params,
					}
					return []string{addr.String()}, true, nil
				}
			}
		}
	}

	// Check for standalone Particl P2CS (cold staking) script: OP_DUP OP_SHA256 <32 bytes> OP_EQUALVERIFY OP_CHECKSIG
	// Format: 0x76 0xa8 0x20 <32-byte-pubkey-hash> 0x88 0xac (total 37 bytes)
	// This differs from standard P2PKH which uses OP_HASH160 (0xa9) with 20 bytes
	if len(script) == 37 &&
		script[0] == 0x76 && // OP_DUP
		script[1] == 0xa8 && // OP_SHA256
		script[2] == 0x20 && // Push 32 bytes
		script[35] == 0x88 && // OP_EQUALVERIFY
		script[36] == 0xac { // OP_CHECKSIG

		// Extract the 32-byte pubkey hash
		pubKeyHash := script[3:35]

		// Encode as Particl P2CS address with version byte 0x39 (produces addresses starting with "2")
		addr := &ParticlP2CSAddress{
			pubKeyHash:  pubKeyHash,
			versionByte: 0x39, // PUBKEY_ADDRESS_256
			params:      p.Params,
		}

		return []string{addr.String()}, true, nil
	}

	// Check for standalone P2SH256 script: OP_SHA256 PUSH32 <32 bytes> OP_EQUAL
	// Format: 0xa8 0x20 <32-byte-script-hash> 0x87 (total 35 bytes)
	// This is Particl's 256-bit version of P2SH (standard P2SH uses HASH160 with 20 bytes)
	if len(script) == 35 &&
		script[0] == 0xa8 && // OP_SHA256
		script[1] == 0x20 && // Push 32 bytes
		script[34] == 0x87 { // OP_EQUAL

		// Extract the 32-byte script hash
		scriptHash := script[2:34]

		// Encode as Particl P2SH256 address with version byte 0x3d (produces addresses starting with "33")
		addr := &ParticlP2CSAddress{
			pubKeyHash:  scriptHash,
			versionByte: 0x3d, // SCRIPT_ADDRESS_256
			params:      p.Params,
		}

		return []string{addr.String()}, true, nil
	}

	// Standard Bitcoin-like addresses (P2PKH, P2SH, P2WPKH, P2WSH, etc.)
	rv, s, _ := p.BitcoinOutputScriptToAddressesFunc(script)
	return rv, s, nil
}

// ParticlP2CSAddress represents a Particl 256-bit hash address (P2CS or P2SH256)
type ParticlP2CSAddress struct {
	pubKeyHash  []byte
	versionByte byte
	params      *chaincfg.Params
}

// String encodes the address as base58check with the specified version byte
func (a *ParticlP2CSAddress) String() string {
	// Encode as base58check: version byte + 32-byte hash
	// Use Sha256D for double-SHA256 checksum (standard Bitcoin-style address encoding)
	encoded := base58.CheckEncode(a.pubKeyHash, []byte{a.versionByte}, base58.Sha256D)
	return encoded
}

// GetAddrDescForUnknownInput returns address descriptor for unknown input
func (p *ParticlParser) GetAddrDescForUnknownInput(tx *bchain.Tx, input int) bchain.AddressDescriptor {
	if len(tx.Vin) > input {
		scriptHex := tx.Vin[input].ScriptSig.Hex

		if scriptHex != "" {
			script, _ := hex.DecodeString(scriptHex)
			return script
		}
	}

	s := make([]byte, 10)
	return s
}
