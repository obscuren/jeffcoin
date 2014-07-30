#define DIFF 1
#define ETH_START_B 2
#define SEED 3
#define JEFF 0

#define BLOCK_F 254
#define BLOCK_N 253
#define ADDR_START 1000

// JeffCoin Genesis :)
contract.storage[BLOCK_N] = 0 
// Set the initial number (so we can track block changes)
contract.storage[ETH_START_B] = block.number()
contract.storage[SEED] = 0
contract.storage[DIFF] = 3
contract.storage[JEFF] = tx.sender()

return compile {
	if tx.sender() == contract.storage[JEFF] {
		if this.data[0] == "die" {
			suicide(contract.storage[JEFF])
		}
	}

	if this.data[0] == "mine" {
		var diff = contract.storage[DIFF]

		var[2] ndat
		ndat[0] = this.data[1]
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
		// Change the seed
		contract.storage[SEED] = contract.storage[SEED] + 1

		// Reward 1000 JeffCoin to the miner
		contract.storage[ADDR_START + tx.sender()] = contract.storage[ADDR_START + tx.sender()] + 100000

		// Check if we need to increase the difficulty
		if contract.storage[ETH_START_B] < block.number() {
			// TODO update difficulty
		}

		stop()
	}
}
