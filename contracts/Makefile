.PHONY: install build test fmt clean

# Install dependencies (not vendored into git — run after cloning)
install:
	forge install OpenZeppelin/openzeppelin-contracts@v5.1.0 --no-git
	forge install foundry-rs/forge-std --no-git

build:
	forge build

test:
	forge test -vvv

fmt:
	forge fmt

clean:
	forge clean
