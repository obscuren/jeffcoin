#define DIFF 1
#define ETH_START_B 2
#define SEED 3

#define BLOCK_F 254
#define BLOCK_N 253

// JeffCoin Genesis :)
this.store[BLOCK_N] = 0 
// Set the initial number (so we can track block changes)
this.store[ETH_START_B] = this.number()
this.store[SEED] = 0
this.store[DIFF] = 1

return compile {
    var diff = this.store[DIFF]
    
    var[2] ndat
    ndat[0] = this.data[0]
    ndat[1] = this.store[SEED]
    m_push(ndat)
    m_push(64)
    asm { sha3 }

    var nonce = m_pop()
    
    // Check if the mined nonce is correct
    for i := 0; i < diff; i++ {
        if byte(nonce, i) != 0 {
            stop() // invalid nonce
        }
    }

    var blockNo = this.store[BLOCK_N]
    // Amount of blocks found for the block (used to determine the difficulty)
    this.store[BLOCK_F + blockNo] = this.store[BLOCK_F + blockNo] + 1

    // Check if we need to increase the difficulty
    if this.store[ETH_START_B] < this.number() {
        // TODO update difficulty
    }
}
