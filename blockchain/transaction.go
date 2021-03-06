package blockchain

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
)

type Transaction struct {
	ID      []byte
	Inputs  []TxInput
	Outputs []TxOutput
}

type TxInput struct {
	ID  []byte
	Out int
	Sig string
}

type TxOutput struct {
	Value  int
	PubKey string
}

func (tx *Transaction) SetID() {
	var encoded bytes.Buffer
	var hash [32]byte
	encoder := gob.NewEncoder(&encoded)
	if err := encoder.Encode(tx); err != nil {
		log.Fatalln(err)
	}
	hash = sha256.Sum256(encoded.Bytes())
	tx.ID = hash[:]
}

func (tx *Transaction) IsCoinbase() bool {
	return len(tx.Inputs) == 1 && len(tx.Inputs[0].ID) == 0 && tx.Inputs[0].Out == -1
}

func (in *TxInput) CanUnlock(data string) bool {
	return in.Sig == data
}

func (out *TxOutput) CanBeUnlocked(data string) bool {
	return out.PubKey == data
}

func CoinbaseTx(to, data string) *Transaction {
	if data == "" {
		data = fmt.Sprintf("Coins to %s", to)
	}
	txIn := TxInput{[]byte{}, -1, data}
	txOut := TxOutput{100, to}
	tx := &Transaction{nil, []TxInput{txIn}, []TxOutput{txOut}}
	tx.SetID()
	return tx
}

func NewTransaction(from, to string, amount int, chain *BlockChain) *Transaction {
	var inputs []TxInput
	var outputs []TxOutput
	acc, validOutputs := chain.FindSpendableOutputs(from, amount)
	if acc < amount {
		log.Fatalln("Not enough funds")
	}
	for txIdx, outs := range validOutputs {
		txID, err := hex.DecodeString(txIdx)
		if err != nil {
			log.Fatalln(err)
		}
		for _, out := range outs {
			input := TxInput{txID, out, from}
			inputs = append(inputs, input)
		}
	}
	outputs = append(outputs, TxOutput{amount, to})
	if acc > amount {
		outputs = append(outputs, TxOutput{acc - amount, from})
	}
	tx := &Transaction{nil, inputs, outputs}
	tx.SetID()
	return tx
}
