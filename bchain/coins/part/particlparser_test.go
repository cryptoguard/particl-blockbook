//go:build unittest

package part

import (
	"encoding/hex"
	"math/big"
	"os"
	"reflect"
	"testing"

	"github.com/martinboehm/btcutil/chaincfg"
	"github.com/trezor/blockbook/bchain"
	"github.com/trezor/blockbook/bchain/coins/btc"
)

func TestMain(m *testing.M) {
	c := m.Run()
	chaincfg.ResetParams()
	os.Exit(c)
}

func TestGetAddrDescFromAddress(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "P2PKH standard",
			args:    args{address: "Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR"},
			want:    "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac",
			wantErr: false,
		},
		{
			name:    "P2CS cold staking",
			args:    args{address: "2urAsyaHCnbMEQLNCCv3unbeysaPpjp1TiWLMWcQRMwYs74qGQh"},
			want:    "76a8200637bcc74ffe834bc66f1e8c5bd6a13bc7a17276338be0016ba8151a2c5473fa88ac",
			wantErr: false,
		},
		{
			name:    "P2SH256 script hash",
			args:    args{address: "33kQnRT9ecMedncHvns3eHbbNzgKPqXMcTCFuWV697J8PiX6myG"},
			want:    "a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd7687",
			wantErr: false,
		},
	}
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.GetAddrDescFromAddress(tt.args.address)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddrDescFromAddress() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			h := hex.EncodeToString(got)
			if !reflect.DeepEqual(h, tt.want) {
				t.Errorf("GetAddrDescFromAddress() = %v, want %v", h, tt.want)
			}
		})
	}
}

func TestGetAddressesFromAddrDesc(t *testing.T) {
	type args struct {
		script string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		want2   bool
		wantErr bool
	}{
		{
			name:    "P2PKH",
			args:    args{script: "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac"},
			want:    []string{"Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR"},
			want2:   true,
			wantErr: false,
		},
		{
			name:    "P2CS (cold staking)",
			args:    args{script: "76a8200637bcc74ffe834bc66f1e8c5bd6a13bc7a17276338be0016ba8151a2c5473fa88ac"},
			want:    []string{"2urAsyaHCnbMEQLNCCv3unbeysaPpjp1TiWLMWcQRMwYs74qGQh"},
			want2:   true,
			wantErr: false,
		},
		{
			name:    "P2SH256",
			args:    args{script: "a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd7687"},
			want:    []string{"33kQnRT9ecMedncHvns3eHbbNzgKPqXMcTCFuWV697J8PiX6myG"},
			want2:   true,
			wantErr: false,
		},
		{
			name:    "Coinstake 66-byte with P2CS",
			args:    args{script: "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac6776a8207be3f09c8d809bc6fa2ced97e35c65d813f29129645bfb45fa3362d28d46123188ac68"},
			want:    []string{"2vjzfpCeiohkBtg3gcQJFLFaV6Yjqehw6Q4SwiKqRwKWfut2zPD"},
			want2:   true,
			wantErr: false,
		},
		{
			name:    "Coinstake 64-byte with P2SH256",
			args:    args{script: "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac67a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd768768"},
			want:    []string{"33kQnRT9ecMedncHvns3eHbbNzgKPqXMcTCFuWV697J8PiX6myG"},
			want2:   true,
			wantErr: false,
		},
	}
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, _ := hex.DecodeString(tt.args.script)
			got, got2, err := parser.GetAddressesFromAddrDesc(b)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddressesFromAddrDesc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAddressesFromAddrDesc() = %v, want %v", got, tt.want)
			}
			if got2 != tt.want2 {
				t.Errorf("GetAddressesFromAddrDesc() got2 = %v, want2 %v", got2, tt.want2)
			}
		})
	}
}

func TestGetAddrDescFromVout(t *testing.T) {
	type args struct {
		vout bchain.Vout
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "P2PKH",
			args:    args{vout: bchain.Vout{ScriptPubKey: bchain.ScriptPubKey{Hex: "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac"}}},
			want:    "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac",
			wantErr: false,
		},
		{
			name:    "P2CS",
			args:    args{vout: bchain.Vout{ScriptPubKey: bchain.ScriptPubKey{Hex: "76a8200637bcc74ffe834bc66f1e8c5bd6a13bc7a17276338be0016ba8151a2c5473fa88ac"}}},
			want:    "76a8200637bcc74ffe834bc66f1e8c5bd6a13bc7a17276338be0016ba8151a2c5473fa88ac",
			wantErr: false,
		},
		{
			name:    "P2SH256",
			args:    args{vout: bchain.Vout{ScriptPubKey: bchain.ScriptPubKey{Hex: "a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd7687"}}},
			want:    "a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd7687",
			wantErr: false,
		},
		{
			name:    "Coinstake 66-byte",
			args:    args{vout: bchain.Vout{ScriptPubKey: bchain.ScriptPubKey{Hex: "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac6776a8207be3f09c8d809bc6fa2ced97e35c65d813f29129645bfb45fa3362d28d46123188ac68"}}},
			want:    "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac6776a8207be3f09c8d809bc6fa2ced97e35c65d813f29129645bfb45fa3362d28d46123188ac68",
			wantErr: false,
		},
		{
			name:    "Coinstake 64-byte",
			args:    args{vout: bchain.Vout{ScriptPubKey: bchain.ScriptPubKey{Hex: "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac67a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd768768"}}},
			want:    "b86376a914912e2b234f941f30b18afbb4fa46171214bf66c888ac67a82016b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd768768",
			wantErr: false,
		},
	}
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.GetAddrDescFromVout(&tt.args.vout)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAddrDescFromVout() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			h := hex.EncodeToString(got)
			if !reflect.DeepEqual(h, tt.want) {
				t.Errorf("GetAddrDescFromVout() = %v, want %v", h, tt.want)
			}
		})
	}
}

func TestParticlP2CSAddress_String(t *testing.T) {
	tests := []struct {
		name        string
		pubKeyHash  string
		versionByte byte
		want        string
	}{
		{
			name:        "P2CS address (version 0x39)",
			pubKeyHash:  "0637bcc74ffe834bc66f1e8c5bd6a13bc7a17276338be0016ba8151a2c5473fa",
			versionByte: 0x39,
			want:        "2urAsyaHCnbMEQLNCCv3unbeysaPpjp1TiWLMWcQRMwYs74qGQh",
		},
		{
			name:        "P2SH256 address (version 0x3d)",
			pubKeyHash:  "16b5038ba914c48cc67b15c980731c7d4628fb2e18591db9058bead591aefd76",
			versionByte: 0x3d,
			want:        "33kQnRT9ecMedncHvns3eHbbNzgKPqXMcTCFuWV697J8PiX6myG",
		},
	}
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pubKeyHash, _ := hex.DecodeString(tt.pubKeyHash)
			addr := &ParticlP2CSAddress{
				pubKeyHash:  pubKeyHash,
				versionByte: tt.versionByte,
				params:      parser.Params,
			}
			got := addr.String()
			if got != tt.want {
				t.Errorf("ParticlP2CSAddress.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseTxFromJson(t *testing.T) {
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	t.Run("Data output transaction", func(t *testing.T) {
		dataOutputJSON := `{
			"txid": "b480399e2dd29d6edd836a57567289d641a42ea2d7608708e734afec06999c1c",
			"version": 672,
			"locktime": 0,
			"vin": [{
				"txid": "547f07ded4809a061d68b867efbeebb7e208c58837a405e92529c655707c6376",
				"vout": 1,
				"scriptSig": {"hex": ""},
				"sequence": 4294967295
			}],
			"vout": [{
				"n": 0,
				"type": "data",
				"data_hex": "52e81e0007f291b9c26409010affffff1e",
				"treasury_fund_cfwd": 269.82893810,
				"smsgfeerate": 0.00000001,
				"smsgdifficulty": "1effffff"
			}, {
				"n": 1,
				"type": "standard",
				"value": 1352.23715049,
				"scriptPubKey": {
					"hex": "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac",
					"address": "Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR",
					"type": "pubkeyhash"
				}
			}]
		}`

		tx, err := parser.ParseTxFromJson([]byte(dataOutputJSON))
		if err != nil {
			t.Errorf("ParseTxFromJson() error = %v", err)
			return
		}

		if len(tx.Vout) < 1 {
			t.Errorf("ParseTxFromJson() vout count = %d, want >= 1", len(tx.Vout))
			return
		}

		if tx.Vout[0].OutputType != "data" {
			t.Errorf("ParseTxFromJson() vout[0].OutputType = %v, want 'data'", tx.Vout[0].OutputType)
		}

		if len(tx.Vout[0].ScriptPubKey.Addresses) != 0 {
			t.Errorf("ParseTxFromJson() vout[0] addresses = %v, want empty", tx.Vout[0].ScriptPubKey.Addresses)
		}

		if len(tx.Vout) > 1 {
			if tx.Vout[1].OutputType != "standard" {
				t.Errorf("ParseTxFromJson() vout[1].OutputType = %v, want 'standard'", tx.Vout[1].OutputType)
			}
			if len(tx.Vout[1].ScriptPubKey.Addresses) != 1 || tx.Vout[1].ScriptPubKey.Addresses[0] != "Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR" {
				t.Errorf("ParseTxFromJson() vout[1] address = %v, want ['Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR']", tx.Vout[1].ScriptPubKey.Addresses)
			}
		}
	})
}

// TestParseBlindTransaction tests parsing of Confidential Transaction (CT) with blinded amounts
// Uses real transaction from Particl mainnet block 2028364
func TestParseBlindTransaction(t *testing.T) {
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	// Real blind (CT) transaction: 4be9ec51111a27794b5c3ea1fe58a2658f58a354595a73d55cffb8394f07ad48
	// Block 2028364, October 2025
	blindTxJSON := `{
		"txid": "4be9ec51111a27794b5c3ea1fe58a2658f58a354595a73d55cffb8394f07ad48",
		"version": 160,
		"locktime": 2028364,
		"vin": [{
			"txid": "184f0ee8668c99918e8357c04304c0d08567020c4bc403993b1aa4b2ae8db7c5",
			"vout": 3,
			"scriptSig": {"hex": ""},
			"sequence": 4294967293
		}],
		"vout": [
			{
				"n": 0,
				"type": "data",
				"data_hex": "06a0910d",
				"ct_fee": 0.00215200
			},
			{
				"n": 1,
				"type": "blind",
				"valueCommitment": "08d120c189c1754c03a20816beb3e9764bf1c28814c86e0e0fef5e96edd9429d39",
				"scriptPubKey": {
					"asm": "OP_DUP OP_HASH160 9f903c425f944b5a1e44da73246b87bc1e03389a OP_EQUALVERIFY OP_CHECKSIG",
					"hex": "76a9149f903c425f944b5a1e44da73246b87bc1e03389a88ac",
					"address": "PnUUNEUgXs99PvQ2cC2KNyYRYwTbnADk2W",
					"type": "pubkeyhash"
				},
				"data_hex": "025e9096196d919512ad7794d4fe0d9208f00a3e2355f0edc77f7a7d0e070d4930",
				"rangeproof": "7dc3ca80691319d1a0181006058b72cb58dcecc2756ed8e3bfa322a2d254786b8e4981945d787cf8fbcc5584ffed57c06b60e10b910655ec1e05382e177892df094d5c0999487aa7e33b6da3dedd12083ea3c0b24ba9eee7ddfb68da09452449f4dbb35daf78d39133cb40f3c9a1296c5097975fb0a3a0a46088528c3450ee9d23b5795cc40b13166f078b5bc50a700274d7618de5664745c99e16640e5736dba9d663f69f00e640b8807ba6fb6b43674b6fead5216e5e50e79b6cabf14cc8e0c2cee0e27c664569e90c9e887b3a16f7529e04fbaeab64a94f60c57a5d4c1352234f58677e8c8627909cea6173037552995e489de4063ad7f16c702d428d8f50fd8e785a9fa59d25f810fa6215e6b9ff443a7fd17a5a3605061095e9953781688be12642020c49e832b47750bd3bd5698b56f3cd9f22181f1e8751f5a61d8ffd04681376864fdb22ca86762a07fe375c51d56e0d4c00b179dbba2a81d4ca603371c700e9c75920a5da2a3f2a09e09a20cd8e3bf22b6b860683a5d3a1957a70d4855528c7908133e8716f571cd0b4c857c84c6c868f42afdf1f205449b28480a1c1dc2431c5a7b1560fd43746a4da5592f86f7b27004af4d92ac3813411f60fb3d233396fe1fae531e38a090f0b85b527071ad55162fda143e343900c474a91d2cf55bef43d6efbf7e3c7bf6a78dfa49fe07f6c2f779f7e850bc58f5969207c1b912cd99188f34be40ace0f0eadef4073f5d4c528f641a5c90896b28e57a0359b52c1deafab7a398a5467e4b1985b2d61363f5e924ad6fe397f596a7a12a40d58fe6a04739bffe9a95bc4764b3752b2f95e7d3428fdb6c300e082dcdff90b6e476b8645a4afd1f312ab1852d185fec808d203bd6f4f2228d90d2bd215ffaa1d0f89181aaac42a880aab2adb493944671f45a7d32779a4a03f91c44dbf70a48a0798e51e"
			}
		]
	}`

	tx, err := parser.ParseTxFromJson([]byte(blindTxJSON))
	if err != nil {
		t.Fatalf("ParseTxFromJson() real blind tx error = %v", err)
	}

	// Verify real transaction was parsed
	if tx.Txid != "4be9ec51111a27794b5c3ea1fe58a2658f58a354595a73d55cffb8394f07ad48" {
		t.Errorf("ParseTxFromJson() txid = %v, want 4be9ec51...", tx.Txid)
	}

	// Verify multiple outputs (data + blind)
	if len(tx.Vout) < 2 {
		t.Fatalf("ParseTxFromJson() blind tx should have at least 2 outputs, got %d", len(tx.Vout))
	}

	// First output is data (CT fee)
	if tx.Vout[0].OutputType != "data" {
		t.Errorf("ParseTxFromJson() vout[0] type = %v, want 'data'", tx.Vout[0].OutputType)
	}

	// Second output is blind
	if tx.Vout[1].OutputType != "blind" {
		t.Errorf("ParseTxFromJson() vout[1] type = %v, want 'blind'", tx.Vout[1].OutputType)
	}

	// Blind outputs show address but amount is hidden
	if len(tx.Vout[1].ScriptPubKey.Addresses) == 0 {
		t.Errorf("ParseTxFromJson() blind output should have address for recipient")
	}
}

// TestParseAnonTransaction tests parsing of RingCT transaction with full privacy
// Uses real transaction from Particl mainnet block 488901
func TestParseAnonTransaction(t *testing.T) {
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	// Real anon (RingCT) transaction: f48d5bce842ac718b2995642ebf2fe35cbe70f10e92069e21d9959dcd6df7384
	// Block 488901, confirmed July 16, 2019
	anonTxJSON := `{
		"txid": "f48d5bce842ac718b2995642ebf2fe35cbe70f10e92069e21d9959dcd6df7384",
		"version": 160,
		"locktime": 0,
		"vin": [{
			"type": "anon",
			"valueSat": -1,
			"num_inputs": 1,
			"ring_size": 5,
			"ring_row_0": "2, 3, 4, 5, 6",
			"sequence": 4294967295
		}],
		"vout": [
			{
				"n": 0,
				"type": "data",
				"data_hex": "06d09f1c",
				"ct_fee": 0.00462800
			},
			{
				"n": 1,
				"type": "anon",
				"pubkey": "02bbc1d7e01e1a6ee800b7dc4990976f1bfd2e6e53541915ce2306bc7e16a2aed6",
				"valueCommitment": "097f3a6f1c56406703393e1d7a52c2ec672cf62a267d72a51b2fed2290f46002f3",
				"data_hex": "033fb674cb4ce2989a3f5a20a46c39c7b8698d210dd85a586ff66264d03166b18f",
				"rangeproof": "d6e91ad3d0cb2a7c0f01a065166ba134769b8306377426be6c0e8372e1c36fc43545ee42399454a6e3c1587b989f88e31a8ab9b3002e84a0fe6bcdbdc2edb0a80d17101ac890ae44bf517e2f1fb0448d902666b6283472386c2bfea41aced9f1ca3aee73220946012a31ea355b0ed8769ba2f952150526a425aea10ac9d9368cb252b6a16f030e4647f80fefa625eacb2718114b8a88989d80a28d6679175f17902668b059a2581209795aaf0c1eb72e83d0d88c09877e6ab701cce6435d226e779c57acae6c7c4e056a2c00ac290770feeace0a503b0c5da55537422dc9979fee9d150de46efd909e37b4d602c0a8053dec0d2a4760ab39cb5a5f11131552e64acaf9c21c913bba64293586406fe4736d7ef6272d1ddc3600532822ccf6c256214a5b4abb78501b9619fe0aff09b284aaf4c11cad32df9c2bb4fdaabbe72bcbdb6133f1c48db30a9f7e091a0d63ae2394783a4589731a8b6e5a6aa0b91c42f402230022037c6a9e13d7b8e3c2cab6fd085c122fad87992240dadb7550dcff4ab07987a8b53fa40edc9c7e8a153a032438e6cc008ddd3a0dd4c21807fc57329e40140550f0fefffa3ae26dfdb3447945959b86d01ca3102f988c76db3065758a1b20ffc4c6234a6b67b2be216448514292e69bbf6ac7e7effece6bc5cbe3884e2844f7b239a67c226597e091c0c11de925a895f68fc5781069a11d136d5cb21fb74d89a08bc90eb20623aa4c628ebff4b69db11b17ee6b96b9d4579b45e2926ca38e52635ff9bf6a1346dcee5c883476e1b330a0cea94fe5c993d62d335528df6306d00772a3659867b91921bda758e261194f0d1422369158a886af397e0287c9ce3aaab536bb055d4fe6f0320f73c016b15ca5d7f8dc76b12c4321e2ed4be9b993af50488d62075d234782969c4c2e454b04dc4b7d567cd097ce3181c3340c0bd87b"
			}
		]
	}`

	tx, err := parser.ParseTxFromJson([]byte(anonTxJSON))
	if err != nil {
		t.Fatalf("ParseTxFromJson() real anon tx error = %v", err)
	}

	// Verify real transaction was parsed
	if tx.Txid != "f48d5bce842ac718b2995642ebf2fe35cbe70f10e92069e21d9959dcd6df7384" {
		t.Errorf("ParseTxFromJson() txid = %v, want f48d5bce...", tx.Txid)
	}

	// Verify anon input type
	if len(tx.Vin) < 1 {
		t.Fatalf("ParseTxFromJson() anon tx has no inputs")
	}

	// Verify multiple outputs (data + anon)
	if len(tx.Vout) < 2 {
		t.Fatalf("ParseTxFromJson() anon tx should have at least 2 outputs, got %d", len(tx.Vout))
	}

	// First output is data (CT fee)
	if tx.Vout[0].OutputType != "data" {
		t.Errorf("ParseTxFromJson() vout[0] type = %v, want 'data'", tx.Vout[0].OutputType)
	}

	// Second output is anon
	if tx.Vout[1].OutputType != "anon" {
		t.Errorf("ParseTxFromJson() vout[1] type = %v, want 'anon'", tx.Vout[1].OutputType)
	}

	// Anon outputs should have no addresses (full privacy)
	if len(tx.Vout[1].ScriptPubKey.Addresses) != 0 {
		t.Errorf("ParseTxFromJson() anon output has addresses = %v, want none for privacy", tx.Vout[1].ScriptPubKey.Addresses)
	}
}

// TestPackTx tests protobuf packing of Particl transactions
func TestPackTx(t *testing.T) {
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	// Use simple P2PKH transaction for testing
	simpleTx := bchain.Tx{
		Hex:       "a002000000000176637c7055c62925e905a43788c508e2b7ebbeef67b8681d069a80d4de077f540100000000ffffffff02041152e81e0007f291b9c26409010affffff1e01e924f67b1f0000001976a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac02473044022004aa8ebef4855db22ea020fa80e36987357fc444b8714f37dc3cd8c4c68f37050220513aa4f756fd8bab89271294b0744b4cfeb237ccd975967d764e84ebd223e5410121038553566dbf0b3c5464d64a8364002ff0b6a087f4c680eca3837dcfa713473ee5",
		Blocktime: 1761309840,
		Txid:      "b480399e2dd29d6edd836a57567289d641a42ea2d7608708e734afec06999c1c",
		LockTime:  0,
		Version:   672,
		Vin: []bchain.Vin{
			{
				ScriptSig: bchain.ScriptSig{
					Hex: "",
				},
				Txid:     "547f07ded4809a061d68b867efbeebb7e208c58837a405e92529c655707c6376",
				Vout:     1,
				Sequence: 4294967295,
			},
		},
		Vout: []bchain.Vout{
			{
				N: 0,
				ScriptPubKey: bchain.ScriptPubKey{
					Hex:       "",
					Addresses: []string{},
				},
			},
			{
				ValueSat: *big.NewInt(135223715049),
				N:        1,
				ScriptPubKey: bchain.ScriptPubKey{
					Hex: "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac",
					Addresses: []string{
						"Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR",
					},
				},
			},
		},
	}

	type args struct {
		tx        bchain.Tx
		height    uint32
		blockTime int64
		parser    *ParticlParser
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "particl-standard-tx",
			args: args{
				tx:        simpleTx,
				height:    2025554,
				blockTime: 1761309840,
				parser:    parser,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.parser.PackTx(&tt.args.tx, tt.args.height, tt.args.blockTime)
			if (err != nil) != tt.wantErr {
				t.Errorf("PackTx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) == 0 {
				t.Errorf("PackTx() returned empty bytes")
			}
			// Verify we can unpack what we packed (round-trip test)
			unpacked, height, err := tt.args.parser.UnpackTx(got)
			if err != nil {
				t.Errorf("UnpackTx() failed to unpack packed transaction: %v", err)
				return
			}
			if height != tt.args.height {
				t.Errorf("UnpackTx() height = %v, want %v", height, tt.args.height)
			}
			if unpacked.Txid != tt.args.tx.Txid {
				t.Errorf("UnpackTx() txid = %v, want %v", unpacked.Txid, tt.args.tx.Txid)
			}
		})
	}
}

// TestUnpackTx tests protobuf unpacking of Particl transactions
func TestUnpackTx(t *testing.T) {
	parser := NewParticlParser(GetChainParams("main"), &btc.Configuration{})

	// First pack a transaction to get valid packed data
	simpleTx := bchain.Tx{
		Hex:       "a002000000000176637c7055c62925e905a43788c508e2b7ebbeef67b8681d069a80d4de077f540100000000ffffffff02041152e81e0007f291b9c26409010affffff1e01e924f67b1f0000001976a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac02473044022004aa8ebef4855db22ea020fa80e36987357fc444b8714f37dc3cd8c4c68f37050220513aa4f756fd8bab89271294b0744b4cfeb237ccd975967d764e84ebd223e5410121038553566dbf0b3c5464d64a8364002ff0b6a087f4c680eca3837dcfa713473ee5",
		Blocktime: 1761309840,
		Txid:      "b480399e2dd29d6edd836a57567289d641a42ea2d7608708e734afec06999c1c",
		LockTime:  0,
		Version:   672,
		Vin: []bchain.Vin{
			{
				ScriptSig: bchain.ScriptSig{
					Hex: "",
				},
				Txid:     "547f07ded4809a061d68b867efbeebb7e208c58837a405e92529c655707c6376",
				Vout:     1,
				Sequence: 4294967295,
			},
		},
		Vout: []bchain.Vout{
			{
				N: 0,
				ScriptPubKey: bchain.ScriptPubKey{
					Hex:       "",
					Addresses: []string{},
				},
			},
			{
				ValueSat: *big.NewInt(135223715049),
				N:        1,
				ScriptPubKey: bchain.ScriptPubKey{
					Hex: "76a914a5cea39a684776fa5d6782faf02baf04251b53bc88ac",
					Addresses: []string{
						"Po3VBGWztKbFnU9rFGKNx2Rtg1zWoS4zTR",
					},
				},
			},
		},
	}

	packedTx, err := parser.PackTx(&simpleTx, 2025554, 1761309840)
	if err != nil {
		t.Fatalf("Failed to pack transaction for test: %v", err)
	}

	type args struct {
		packedTx []byte
		parser   *ParticlParser
	}
	tests := []struct {
		name      string
		args      args
		wantTxid  string
		wantHeight uint32
		wantErr   bool
	}{
		{
			name: "particl-standard-tx",
			args: args{
				packedTx: packedTx,
				parser:   parser,
			},
			wantTxid:   "b480399e2dd29d6edd836a57567289d641a42ea2d7608708e734afec06999c1c",
			wantHeight: 2025554,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, height, err := tt.args.parser.UnpackTx(tt.args.packedTx)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnpackTx() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if height != tt.wantHeight {
				t.Errorf("UnpackTx() height = %v, want %v", height, tt.wantHeight)
			}
			if got.Txid != tt.wantTxid {
				t.Errorf("UnpackTx() txid = %v, want %v", got.Txid, tt.wantTxid)
			}
			// Verify transaction structure
			if len(got.Vin) == 0 {
				t.Errorf("UnpackTx() transaction has no inputs")
			}
			if len(got.Vout) == 0 {
				t.Errorf("UnpackTx() transaction has no outputs")
			}
		})
	}
}
