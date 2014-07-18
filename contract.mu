#define DIFF 1
#define ETH_START_B 2
#define SEED 3

#define BLOCK_F 254
#define BLOCK_N 253

// JeffCoin Genesis :)
contract.storage[BLOCK_N] = 0 
// Set the initial number (so we can track block changes)
contract.storage[ETH_START_B] = block.number()
contract.storage[SEED] = 0
contract.storage[DIFF] = 1

return compile {
    var diff = contract.storage[DIFF]
    
    var[2] ndat
    ndat[0] = this.data[0]
    ndat[1] = contract.storage[SEED]

    var nonce = sha3(ndat, sizeof(ndat))
    
    // Check if the mined nonce is correct
    for i := 0; i < diff; i++ {
        if byte(nonce, i) != 0 {
            stop() // invalid nonce
        }
    }

    var blockNo = contract.storage[BLOCK_N]
    // Amount of blocks found for the block (used to determine the difficulty)
    contract.storage[BLOCK_F + blockNo] = contract.storage[BLOCK_F + blockNo] + 1

    // Check if we need to increase the difficulty
    if contract.storage[ETH_START_B] < block.number() {
        // TODO update difficulty
    }
}
