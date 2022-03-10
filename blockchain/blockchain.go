package blockchain

import (
	"fmt"
	"github.com/dgraph-io/badger/v3"
	"log"
)

const (
	dbPath = "./tmp/blocks"
)

type BlockChain struct {
	LastHash []byte
	Database *badger.DB
}

type BlockChainIterator struct {
	CurrentHash []byte
	Database    *badger.DB
}

func (chain *BlockChain) AddBlock(data string) {
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
	newBlock := CreateBlock(data, lastHash)
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

func InitBlockChain() *BlockChain {
	var lastHash []byte
	opts := badger.DefaultOptions(dbPath)
	opts.Logger = nil
	db, err := badger.Open(opts)
	if err != nil {
		log.Fatalln(err)
	}
	err = db.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte("lh")); err == badger.ErrKeyNotFound {
			fmt.Println("No existing blockchain found")
			genesis := Genesis()
			fmt.Println("Genesis proved")
			if err = txn.Set(genesis.Hash, genesis.Serialize()); err != nil {
				log.Fatalln(err)
			}
			err = txn.Set([]byte("lh"), genesis.Hash)
			lastHash = genesis.Hash
			return err
		}
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
