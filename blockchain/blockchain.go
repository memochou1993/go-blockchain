package blockchain

import (
	"encoding/hex"
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"log"
	"os"
	"runtime"
)

const (
	dbPath      = "./tmp/blocks"
	dbFile      = "./tmp/blocks/MANIFEST"
	genesisData = "First Transaction from Genesis"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *BlockChain) AddBlock(transactions []*Transaction) {
	var lastHash []byte
	err := chain.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			log.Fatalln(err)
		}
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}
	newBlock := CreateBlock(transactions, lastHash)
	err = chain.Database.Update(func(txn *badger.Txn) error {
		if err := txn.Set(newBlock.Hash, newBlock.Serialize()); err != nil {
			log.Fatalln(err)
		}
		err = txn.Set([]byte("lh"), newBlock.Hash)
		chain.LastHash = newBlock.Hash
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func (chain *BlockChain) Iterator() *BlockChainIterator {
	return &BlockChainIterator{chain.LastHash, chain.Database}
}

func (iter *BlockChainIterator) Next() *Block {
	var block *Block
	err := iter.Database.View(func(txn *badger.Txn) error {
		item, err := txn.Get(iter.CurrentHash)
		encodedBlock, err := item.ValueCopy(nil)
		block = Deserialize(encodedBlock)
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}
	iter.CurrentHash = block.PrevHash
	return block
}

func (chain *BlockChain) FindUnspentTransactions(address string) []Transaction {
	var unspentTxs []Transaction
	spentTxOutputs := make(map[string][]int)
	iter := chain.Iterator()
	for {
		block := iter.Next()
		for _, tx := range block.Transactions {
			txID := hex.EncodeToString(tx.ID)
		Outputs:
			for outIdx, out := range tx.Outputs {
				if spentTxOutputs[txID] != nil {
					for _, spentOut := range spentTxOutputs[txID] {
						if spentOut == outIdx {
							continue Outputs
						}
					}
				}
				if out.CanBeUnlocked(address) {
					unspentTxs = append(unspentTxs, *tx)
				}
			}
			if !tx.IsCoinbase() {
				for _, in := range tx.Inputs {
					if in.CanUnlock(address) {
						inTxID := hex.EncodeToString(in.ID)
						spentTxOutputs[inTxID] = append(spentTxOutputs[inTxID], in.Out)
					}
				}
			}
		}
		if len(block.PrevHash) == 0 {
			break
		}
	}
	return unspentTxs
}

func (chain *BlockChain) FindUnspentTxOutputs(address string) []TxOutput {
	var unspentTxOutputs []TxOutput
	unspentTransactions := chain.FindUnspentTransactions(address)
	for _, tx := range unspentTransactions {
		for _, out := range tx.Outputs {
			if out.CanBeUnlocked(address) {
				unspentTxOutputs = append(unspentTxOutputs, out)
			}
		}
	}
	return unspentTxOutputs
}

func (chain *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {
	unspentOutputs := make(map[string][]int)
	unspentTxs := chain.FindUnspentTransactions(address)
	accumulated := 0
Work:
	for _, tx := range unspentTxs {
		txID := hex.EncodeToString(tx.ID)
		for outIdx, out := range tx.Outputs {
			if out.CanBeUnlocked(address) && accumulated < amount {
				accumulated += out.Value
				unspentOutputs[txID] = append(unspentOutputs[txID], outIdx)
				if accumulated >= amount {
					break Work
				}
			}
		}
	}
	return accumulated, unspentOutputs
}

func InitBlockChain(address string) *BlockChain {
	var lastHash []byte
	if DatabaseExists() {
		fmt.Println("Blockchain already exists")
		runtime.Goexit()
	}
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.Update(func(txn *badger.Txn) error {
		cbTx := CoinbaseTx(address, genesisData)
		genesis := Genesis(cbTx)
		fmt.Println("Genesis proved")
		if err = txn.Set(genesis.Hash, genesis.Serialize()); err != nil {
			log.Fatalln(err)
		}
		err = txn.Set([]byte("lh"), genesis.Hash)
		lastHash = genesis.Hash
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &BlockChain{lastHash, db}
}

func ContinueBlockChain(address string) *BlockChain {
	if !DatabaseExists() {
		fmt.Println("No existing blockchain found, create one!")
	}
	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	err = db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("lh"))
		if err != nil {
			log.Fatalln(err)
		}
		lastHash, err = item.ValueCopy(nil)
		return err
	})
	if err != nil {
		log.Fatalln(err)
	}
	return &BlockChain{lastHash, db}
}

func DatabaseExists() bool {
	if _, err := os.Stat(dbFile); os.IsNotExist(err) {
		return false
	}
	return true
}
