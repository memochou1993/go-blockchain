package main

import (
	"flag"
	"fmt"
	"github.com/memochou1993/go-blockchain/blockchain"
	"log"
	"os"
	"runtime"
	"strconv"
)

type CommandLine struct{}

func (cli *CommandLine) printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  get-balance -address ADDRESS - gets the balance for the address")
	fmt.Println("  create-blockchain -address ADDRESS - creates a blockchain and sends genesis reward to address")
	fmt.Println("  print-chain - prints the blocks in the chain")
	fmt.Println("  send -from FROM -to TO -amount AMOUNT - sends amount of coins")
}

func (cli *CommandLine) validateArgs() {
	if len(os.Args) < 2 {
		cli.printUsage()
		runtime.Goexit()
	}
}

func (cli *CommandLine) printChain() {
	chain := blockchain.ContinueBlockChain("")
	defer chain.Database.Close()
	iter := chain.Iterator()
	for {
		block := iter.Next()
		fmt.Printf("Previous Hash: %x\n", block.PrevHash)
		fmt.Printf("Hash: %x\n", block.Hash)
		pow := blockchain.NewProof(block)
		fmt.Printf("Pow: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
		if len(block.PrevHash) == 0 {
			break
		}
	}
}

func (cli *CommandLine) createBlockChain(address string) {
	chain := blockchain.InitBlockChain(address)
	chain.Database.Close()
	fmt.Println("Finished!")
}

func (cli *CommandLine) getBalance(address string) {
	chain := blockchain.ContinueBlockChain(address)
	defer chain.Database.Close()
	balance := 0
	unspentTxOutputs := chain.FindUnspentTxOutputs(address)
	for _, out := range unspentTxOutputs {
		balance += out.Value
	}
	fmt.Printf("Balance of %s: %d\n", address, balance)
}

func (cli *CommandLine) send(from, to string, amount int) {
	chain := blockchain.ContinueBlockChain(from)
	defer chain.Database.Close()
	tx := blockchain.NewTransaction(from, to, amount, chain)
	chain.AddBlock([]*blockchain.Transaction{tx})
	fmt.Println("Success!")
}

func (cli *CommandLine) run() {
	cli.validateArgs()
	getBalanceCmd := flag.NewFlagSet("get-balance", flag.ExitOnError)
	createBlockchainCmd := flag.NewFlagSet("create-blockchain", flag.ExitOnError)
	sendCmd := flag.NewFlagSet("send", flag.ExitOnError)
	printChainCmd := flag.NewFlagSet("print-chain", flag.ExitOnError)
	getBalanceAddress := getBalanceCmd.String("address", "", "The address to get balance for")
	createBlockchainAddress := createBlockchainCmd.String("address", "", "The address to send genesis block reward to")
	sendFrom := sendCmd.String("from", "", "Source wallet address")
	sendTo := sendCmd.String("to", "", "Destination wallet address")
	sendAmount := sendCmd.Int("amount", 0, "Amount to send")
	switch os.Args[1] {
	case "get-balance":
		err := getBalanceCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "create-blockchain":
		err := createBlockchainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "print-chain":
		err := printChainCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	case "send":
		err := sendCmd.Parse(os.Args[2:])
		if err != nil {
			log.Panic(err)
		}
	default:
		cli.printUsage()
		runtime.Goexit()
	}
	if getBalanceCmd.Parsed() {
		if *getBalanceAddress == "" {
			getBalanceCmd.Usage()
			runtime.Goexit()
		}
		cli.getBalance(*getBalanceAddress)
	}
	if createBlockchainCmd.Parsed() {
		if *createBlockchainAddress == "" {
			createBlockchainCmd.Usage()
			runtime.Goexit()
		}
		cli.createBlockChain(*createBlockchainAddress)
	}
	if printChainCmd.Parsed() {
		cli.printChain()
	}
	if sendCmd.Parsed() {
		if *sendFrom == "" || *sendTo == "" || *sendAmount <= 0 {
			sendCmd.Usage()
			runtime.Goexit()
		}
		cli.send(*sendFrom, *sendTo, *sendAmount)
	}
}

func main() {
	defer os.Exit(0)
	cli := CommandLine{}
	cli.run()
}
