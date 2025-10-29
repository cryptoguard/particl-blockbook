#!/bin/bash
# test_ui.sh - UI/Template regression tests for Particl Blockbook integration
#
# Tests web interface display of all Particl transaction types:
# - Confidential Transactions (CT/Blind)
# - RingCT (Anonymous) transactions
# - Standard transactions
# - Cold Staking (P2CS) transactions
# - Address pages and API endpoints
#
# Location: bchain/coins/part/test_ui.sh
# Run from any directory: bash bchain/coins/part/test_ui.sh
# Requires: Blockbook running on http://localhost:9131

BLOCKBOOK_URL="http://localhost:9131"

echo "========================================"
echo "Particl Blockbook Comprehensive Test Suite"
echo "========================================"
echo "Testing against: $BLOCKBOOK_URL"
echo ""

PASS_COUNT=0
FAIL_COUNT=0
PENDING_COUNT=0

# Test 1: Confidential Transaction (CT/Blind)
echo "Test 1: Confidential Transaction (CT/Blind)"
echo "--------------------------------------------"
BLIND_TX="4be9ec51111a27794b5c3ea1fe58a2658f58a354595a73d55cffb8394f07ad48"
echo "Transaction: $BLIND_TX"
echo "Block: 2,028,364"

# Check for "Blinded" text
BLIND_COUNT=$(curl -s "$BLOCKBOOK_URL/tx/$BLIND_TX" | grep -o "Blinded" | wc -l)
echo "  - 'Blinded' occurrences: $BLIND_COUNT (expected: 4)"

# Check that "0 PART" only appears for data output (not for blind amounts)
ZERO_PART_COUNT=$(curl -s "$BLOCKBOOK_URL/tx/$BLIND_TX" | grep -c ">0 PART<")
echo "  - '0 PART' occurrences: $ZERO_PART_COUNT (expected: 1 for data output)"

# Verify CT fee is displayed
CT_FEE=$(curl -s "$BLOCKBOOK_URL/tx/$BLIND_TX" | grep -o "0\.00215[0-9]*" | head -1)
echo "  - CT Fee displayed: $CT_FEE PART (expected: 0.00215199)"

if [ "$BLIND_COUNT" -eq 4 ] && [ "$ZERO_PART_COUNT" -eq 1 ]; then
    echo "✅ PASS: CT transaction displays correctly"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: CT transaction display incorrect"
    echo "   Expected: Blinded=4, 0 PART=1"
    echo "   Got: Blinded=$BLIND_COUNT, 0 PART=$ZERO_PART_COUNT"
    ((FAIL_COUNT++))
fi
echo ""

# Test 2: Anonymous Transaction (RingCT)
echo "Test 2: Anonymous Transaction (RingCT)"
echo "--------------------------------------------"
ANON_TX="f48d5bce842ac718b2995642ebf2fe35cbe70f10e92069e21d9959dcd6df7384"
echo "Transaction: $ANON_TX"
echo "Block: 488,901"

# Check for "Anonymous" label
ANON_LABEL=$(curl -s "$BLOCKBOOK_URL/tx/$ANON_TX" | grep -c "Anonymous")
echo "  - 'Anonymous' label: $ANON_LABEL (expected: >= 1)"

# Check for "Blinded" amounts
ANON_BLIND=$(curl -s "$BLOCKBOOK_URL/tx/$ANON_TX" | grep -o "Blinded" | wc -l)
echo "  - 'Blinded' occurrences: $ANON_BLIND (expected: >= 2)"

# Check for ring size info
RING_SIZE=$(curl -s "$BLOCKBOOK_URL/tx/$ANON_TX" | grep -o "ring" | head -1)
echo "  - Ring size mentioned: $([ -n "$RING_SIZE" ] && echo "Yes" || echo "No") (expected: Yes)"

# Verify CT fee for anon tx
ANON_FEE=$(curl -s "$BLOCKBOOK_URL/tx/$ANON_TX" | grep -o "0\.00462[0-9]*" | head -1)
echo "  - CT Fee displayed: $ANON_FEE PART (expected: 0.00462800)"

if [ "$ANON_LABEL" -ge 1 ] && [ "$ANON_BLIND" -ge 2 ]; then
    echo "✅ PASS: RingCT transaction displays correctly"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: RingCT transaction display incorrect"
    echo "   Expected: Anonymous>=1, Blinded>=2"
    echo "   Got: Anonymous=$ANON_LABEL, Blinded=$ANON_BLIND"
    ((FAIL_COUNT++))
fi
echo ""

# Test 3: Address Page Load Test
echo "Test 3: Address Page Loading"
echo "--------------------------------------------"
TEST_ADDR="PnUUNEUgXs99PvQ2cC2KNyYRYwTbnADk2W"
echo "Address: $TEST_ADDR (from CT tx)"

ADDR_RESPONSE=$(curl -s "$BLOCKBOOK_URL/address/$TEST_ADDR")
if echo "$ADDR_RESPONSE" | grep -q "$TEST_ADDR"; then
    echo "  - Address page loads: Yes"

    # Check for balance display
    if echo "$ADDR_RESPONSE" | grep -q -E "Balance|Total"; then
        echo "  - Balance section present: Yes"
    fi

    # Check for transaction list
    if echo "$ADDR_RESPONSE" | grep -q "tx-detail\|Transaction"; then
        echo "  - Transaction list present: Yes"
    fi

    echo "✅ PASS: Address page loads correctly"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: Address page failed to load"
    ((FAIL_COUNT++))
fi
echo ""

# Test 4: Standard Transaction (PoS Coinbase)
echo "Test 4: Standard (SegWit) Transaction"
echo "--------------------------------------------"
STD_TX="82fa17dfb9c5fb91d8fcc89674756c5df86d743ca7d20e58977926a648fb37f8"
echo "Transaction: $STD_TX"
echo "Block: 2,028,000 (PoS Coinbase)"

# Check that NO "Blinded" appears (all amounts visible)
STD_BLIND=$(curl -s "$BLOCKBOOK_URL/tx/$STD_TX" | grep -c "Blinded")
echo "  - 'Blinded' occurrences: $STD_BLIND (expected: 0)"

# Check for actual PART amounts
STD_AMOUNTS=$(curl -s "$BLOCKBOOK_URL/tx/$STD_TX" | grep -o "[0-9]\+\.[0-9]* PART" | wc -l)
echo "  - PART amounts displayed: $STD_AMOUNTS (expected: >= 3)"

# Verify specific large amount is shown
if curl -s "$BLOCKBOOK_URL/tx/$STD_TX" | grep -q "1139\."; then
    echo "  - Large amount visible: Yes (1139.45... PART)"
else
    echo "  - Large amount visible: No"
fi

if [ "$STD_BLIND" -eq 0 ] && [ "$STD_AMOUNTS" -ge 3 ]; then
    echo "✅ PASS: Standard transaction shows visible amounts"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: Standard transaction not displaying correctly"
    echo "   Expected: Blinded=0, Amounts>=3"
    echo "   Got: Blinded=$STD_BLIND, Amounts=$STD_AMOUNTS"
    ((FAIL_COUNT++))
fi
echo ""

# Test 5: Cold Staking Transaction (P2CS)
echo "Test 5: Cold Staking (P2CS) Transaction"
echo "--------------------------------------------"
P2CS_TX="819e21b3b1d12539371df023b0865da74648f342c912b535f639ca622d97abf5"
echo "Transaction: $P2CS_TX"
echo "Block: 2,027,702"

# Check that NO "Blinded" appears (cold staking uses visible amounts)
P2CS_BLIND=$(curl -s "$BLOCKBOOK_URL/tx/$P2CS_TX" | grep -c "Blinded")
echo "  - 'Blinded' occurrences: $P2CS_BLIND (expected: 0)"

# Check for actual PART amounts (10 inputs + 32 outputs)
P2CS_AMOUNTS=$(curl -s "$BLOCKBOOK_URL/tx/$P2CS_TX" | grep -o "[0-9]\+\.[0-9]* PART" | wc -l)
echo "  - PART amounts displayed: $P2CS_AMOUNTS (expected: >= 40)"

# Verify P2CS addresses with "2" prefix are displayed
P2CS_ADDR_2=$(curl -s "$BLOCKBOOK_URL/tx/$P2CS_TX" | grep -o "2[uvw][A-Za-z0-9]\{40,\}" | head -1)
if [ -n "$P2CS_ADDR_2" ]; then
    echo "  - P2CS addresses (prefix '2'): Found (${P2CS_ADDR_2:0:10}...)"
else
    echo "  - P2CS addresses (prefix '2'): Not found"
fi

# Verify P2CS addresses with "33" prefix are displayed
P2CS_ADDR_33=$(curl -s "$BLOCKBOOK_URL/tx/$P2CS_TX" | grep -o "33[A-Za-z0-9]\{40,\}" | head -1)
if [ -n "$P2CS_ADDR_33" ]; then
    echo "  - P2CS addresses (prefix '33'): Found (${P2CS_ADDR_33:0:10}...)"
else
    echo "  - P2CS addresses (prefix '33'): Not found"
fi

# Verify standard P2PKH change address is displayed
P2CS_PKH=$(curl -s "$BLOCKBOOK_URL/tx/$P2CS_TX" | grep -o "Pk[A-Za-z0-9]\{30,\}" | head -1)
if [ -n "$P2CS_PKH" ]; then
    echo "  - Standard P2PKH address: Found (${P2CS_PKH:0:10}...)"
else
    echo "  - Standard P2PKH address: Not found"
fi

if [ "$P2CS_BLIND" -eq 0 ] && [ "$P2CS_AMOUNTS" -ge 40 ] && [ -n "$P2CS_ADDR_2" ] && [ -n "$P2CS_ADDR_33" ]; then
    echo "✅ PASS: Cold staking transaction displays correctly"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: Cold staking transaction not displaying correctly"
    echo "   Expected: Blinded=0, Amounts>=40, Both address types present"
    echo "   Got: Blinded=$P2CS_BLIND, Amounts=$P2CS_AMOUNTS"
    ((FAIL_COUNT++))
fi
echo ""

# Test 6: API Endpoint Test
echo "Test 6: API Endpoint Tests"
echo "--------------------------------------------"

# Test TX API endpoint
API_TX=$(curl -s "$BLOCKBOOK_URL/api/v2/tx/$BLIND_TX" | python3 -c "import sys, json; tx=json.load(sys.stdin); print(tx.get('txid', 'NONE'))" 2>/dev/null || echo "ERROR")
echo "  - API TX endpoint: $([ "$API_TX" = "$BLIND_TX" ] && echo "✅ Working" || echo "❌ Failed")"

# Test block endpoint
API_BLOCK=$(curl -s "$BLOCKBOOK_URL/api/v2/block/2028364" | python3 -c "import sys, json; b=json.load(sys.stdin); print(b.get('height', 'NONE'))" 2>/dev/null || echo "ERROR")
echo "  - API Block endpoint: $([ "$API_BLOCK" = "2028364" ] && echo "✅ Working" || echo "❌ Failed")"

if [ "$API_TX" = "$BLIND_TX" ] && [ "$API_BLOCK" = "2028364" ]; then
    echo "✅ PASS: API endpoints working"
    ((PASS_COUNT++))
else
    echo "❌ FAIL: API endpoints not responding correctly"
    ((FAIL_COUNT++))
fi
echo ""

# Summary
echo "========================================"
echo "Test Summary"
echo "========================================"
echo "✅ PASSED: $PASS_COUNT"
echo "❌ FAILED: $FAIL_COUNT"
echo "⏳ PENDING: $PENDING_COUNT"
echo ""

TOTAL=$((PASS_COUNT + FAIL_COUNT + PENDING_COUNT))
if [ $TOTAL -gt 0 ]; then
    SUCCESS_RATE=$((PASS_COUNT * 100 / (PASS_COUNT + FAIL_COUNT)))
    echo "Success Rate: $SUCCESS_RATE% (excluding pending tests)"
fi
echo ""

if [ $FAIL_COUNT -eq 0 ]; then
    echo "🎉 All implemented tests PASSED!"
    exit 0
else
    echo "⚠️  Some tests FAILED - review output above"
    exit 1
fi
