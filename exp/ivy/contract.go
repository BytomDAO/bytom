package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/bytom/exp/ivy/instance"
	"github.com/bytom/protocol/bc"
	"github.com/spf13/cobra"
)

// the TimeLayout by time Template
const (
	TimeLayout string = "2006-01-02 15:05:05"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	Execute()
}

// IvyCmd is ivyInstance's root command.
var IvyCmd = &cobra.Command{
	Use:   "ivy",
	Short: "ivy is a generate contract program tools",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
		}
	},
}

// Execute adds all child commands to the root command IvyCmd and sets flags appropriately.
func Execute() {
	AddCommands()

	if _, err := IvyCmd.ExecuteC(); err != nil {
		os.Exit(0)
	}
}

// AddCommands adds child commands to the root command IvyCmd.
func AddCommands() {
	IvyCmd.AddCommand(cmdLockWithPublicKey)
	IvyCmd.AddCommand(cmdLockWithMultiSig)
	IvyCmd.AddCommand(cmdLockWithPublicKeyHash)
	IvyCmd.AddCommand(cmdRevealPreimage)
	IvyCmd.AddCommand(cmdTradeOffer)
	IvyCmd.AddCommand(cmdEscrow)
	IvyCmd.AddCommand(cmdLoanCollateral)
	IvyCmd.AddCommand(cmdCallOption)
}

var cmdLockWithPublicKey = &cobra.Command{
	Use:   "LockWithPublicKey <pubkey>",
	Short: "create a new contract for LockWithPublicKey",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pubkeyStr := args[0]
		if len(pubkeyStr) != 64 {
			fmt.Printf("the length of byte pubkey[%d] is not equal 64\n", len(pubkeyStr))
			os.Exit(0)
		}

		pubkey, _ := hex.DecodeString(pubkeyStr)
		contractProgram, err := instance.PayToLockWithPublicKey(pubkey)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToLockWithPublicKey(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdLockWithMultiSig = &cobra.Command{
	Use:   "LockWithMultiSig [pubkey1] [pubkey2] [pubkey3]",
	Short: "create a new contract for LockWithMultiSig",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		pubkeyStr1 := args[0]
		pubkeyStr2 := args[1]
		pubkeyStr3 := args[2]
		if len(pubkeyStr1) != 64 || len(pubkeyStr2) != 64 || len(pubkeyStr3) != 64 {
			fmt.Printf("the length of byte pubkey1[%d] or pubkey2[%d] or pubkey3[%d] is not equal 64\n",
				len(pubkeyStr1), len(pubkeyStr2), len(pubkeyStr3))
			os.Exit(0)
		}

		pubkey1, _ := hex.DecodeString(pubkeyStr1)
		pubkey2, _ := hex.DecodeString(pubkeyStr2)
		pubkey3, _ := hex.DecodeString(pubkeyStr3)
		contractProgram, err := instance.PayToLockWithMultiSig(pubkey1, pubkey2, pubkey3)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToLockWithMultiSig(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdLockWithPublicKeyHash = &cobra.Command{
	Use:   "LockWithPublicKeyHash <pubkeyHash>",
	Short: "create a new contract for LockWithPublicKeyHash",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		pubkeyHashStr := args[0]
		if len(pubkeyHashStr) != 64 {
			fmt.Printf("the length of byte pubkeyHash[%d] is not equal 64\n", len(pubkeyHashStr))
			os.Exit(0)
		}

		pubkeyHash, _ := hex.DecodeString(pubkeyHashStr)
		contractProgram, err := instance.PayToLockWithPublicKeyHash(pubkeyHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToLockWithPublicKeyHash(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdRevealPreimage = &cobra.Command{
	Use:   "RevealPreimage <valueHash>",
	Short: "create a new contract for RevealPreimage",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		valueHashStr := args[0]
		if len(valueHashStr) != 64 {
			fmt.Printf("the length of byte valueHash[%d] is not equal 64\n", len(valueHashStr))
			os.Exit(0)
		}

		valueHash, _ := hex.DecodeString(valueHashStr)
		contractProgram, err := instance.PayToRevealPreimage(valueHash)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToRevealPreimage(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdTradeOffer = &cobra.Command{
	Use:   "TradeOffer [assetID] [amount] [seller] [pubkey]",
	Short: "create a new contract for TradeOffer",
	Args:  cobra.ExactArgs(4),
	Run: func(cmd *cobra.Command, args []string) {
		assetStr := args[0]
		amountStr := args[1]
		sellerStr := args[2]
		pubkeyStr := args[3]
		if len(assetStr) != 64 || len(pubkeyStr) != 64 {
			fmt.Printf("the length of byte assetID[%d] or pubkey[%d] is not equal 64\n", len(assetStr), len(pubkeyStr))
			os.Exit(0)
		}

		assetByte, _ := hex.DecodeString(assetStr)
		var b [32]byte
		copy(b[:], assetByte[:32])
		assetID := bc.NewAssetID(b)

		amount, err := strconv.ParseUint(amountStr, 10, 64)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		seller, err := hex.DecodeString(sellerStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		pubkey, _ := hex.DecodeString(pubkeyStr)

		contractProgram, err := instance.PayToTradeOffer(assetID, amount, seller, pubkey)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToTradeOffer(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdEscrow = &cobra.Command{
	Use:   "Escrow [pubkey] [sender] [recipient]",
	Short: "create a new contract for Escrow",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		pubkeyStr := args[0]
		senderStr := args[1]
		recipientStr := args[2]
		if len(pubkeyStr) != 64 {
			fmt.Printf("the length of byte pubkey[%d] is not equal 64\n", len(pubkeyStr))
			os.Exit(0)
		}

		pubkey, _ := hex.DecodeString(pubkeyStr)
		sender, err := hex.DecodeString(senderStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		recipient, err := hex.DecodeString(recipientStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		contractProgram, err := instance.PayToEscrow(pubkey, sender, recipient)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToEscrow(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdLoanCollateral = &cobra.Command{
	Use:   "LoanCollateral [assetID] [amount] [dueTime] [lender] [borrower]",
	Short: "create a new contract for LoanCollateral",
	Args:  cobra.ExactArgs(5),
	Run: func(cmd *cobra.Command, args []string) {
		assetStr := args[0]
		amountStr := args[1]
		dueTimeStr := args[2]
		lenderStr := args[3]
		borrowerStr := args[4]
		if len(assetStr) != 64 {
			fmt.Printf("the length of byte assetID[%d] is not equal 64\n", len(assetStr))
			os.Exit(0)
		}

		assetByte, _ := hex.DecodeString(assetStr)
		var b [32]byte
		copy(b[:], assetByte[:32])
		assetID := bc.NewAssetID(b)

		amount, err := strconv.ParseUint(amountStr, 10, 64)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		dueTimeStr = strings.Replace(dueTimeStr, "*", " ", -1)
		loc, _ := time.LoadLocation("Local")
		dueTime, err := time.ParseInLocation(TimeLayout, dueTimeStr, loc)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		lender, err := hex.DecodeString(lenderStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		borrower, err := hex.DecodeString(borrowerStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		contractProgram, err := instance.PayToLoanCollateral(assetID, amount, dueTime, lender, borrower)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToLoanCollateral(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}

var cmdCallOption = &cobra.Command{
	Use:   "CallOption [amountPrice] [assetID] [seller] [buyerPubkey] [deadline]",
	Short: "create a new contract for CallOption",
	Args:  cobra.ExactArgs(5),
	Run: func(cmd *cobra.Command, args []string) {
		amountPriceStr := args[0]
		assetStr := args[1]
		sellerStr := args[2]
		buyerPubkeyStr := args[3]
		deadlineStr := args[4]
		if len(assetStr) != 64 || len(buyerPubkeyStr) != 64 {
			fmt.Printf("the length of byte assetID[%d] or buyerPubkey[%d] is not equal 64\n", len(assetStr), len(buyerPubkeyStr))
			os.Exit(0)
		}

		amountPrice, err := strconv.ParseUint(amountPriceStr, 10, 64)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		assetByte, _ := hex.DecodeString(assetStr)
		var b [32]byte
		copy(b[:], assetByte[:32])
		assetID := bc.NewAssetID(b)

		seller, err := hex.DecodeString(sellerStr)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		buyerPubkey, _ := hex.DecodeString(buyerPubkeyStr)

		deadlineStr = strings.Replace(deadlineStr, "*", " ", -1)
		loc, _ := time.LoadLocation("Local")
		deadline, err := time.ParseInLocation(TimeLayout, deadlineStr, loc)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		contractProgram, err := instance.PayToCallOption(amountPrice, assetID, seller, buyerPubkey, deadline)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		//check the program
		if _, err := instance.ParsePayToCallOption(contractProgram); err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		fmt.Printf("The Result ControlProgram:\n%s\n", hex.EncodeToString(contractProgram))
	},
}
