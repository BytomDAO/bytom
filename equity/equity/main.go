package main

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"os"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/bytom/bytom/equity/compiler"
	equityutil "github.com/bytom/bytom/equity/equity/util"
)

const (
	strBin      string = "bin"
	strShift    string = "shift"
	strInstance string = "instance"
)

var (
	bin      = false
	shift    = false
	instance = false
)

func init() {
	equityCmd.PersistentFlags().BoolVar(&bin, strBin, false, "Binary of the contracts in hex.")
	equityCmd.PersistentFlags().BoolVar(&shift, strShift, false, "Function shift of the contracts.")
	equityCmd.PersistentFlags().BoolVar(&instance, strInstance, false, "Object of the Instantiated contracts.")
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	if err := equityCmd.Execute(); err != nil {
		os.Exit(0)
	}
}

var equityCmd = &cobra.Command{
	Use:     "equity <input_file>",
	Short:   "equity commandline compiler",
	Example: "equity contract_name [contract_args...] --bin --instance",
	Args:    cobra.RangeArgs(1, 100),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) < 1 {
			cmd.Usage()
		}

		contractFile, err := os.Open(args[0])
		if err != nil {
			fmt.Printf("An error [%v] occurred on opening the file, please check whether the file exists or can be accessed.\n", err)
			os.Exit(0)
		}
		defer contractFile.Close()

		reader := bufio.NewReader(contractFile)
		contracts, err := compiler.Compile(reader)
		if err != nil {
			fmt.Println("Compile contract failed:", err)
			os.Exit(0)
		}

		if len(contracts) == 0 {
			fmt.Println("The contract is empty!")
			os.Exit(0)
		}

		// Print the result for all contracts
		for i, contract := range contracts {
			fmt.Printf("======= %v =======\n", contract.Name)
			if bin {
				fmt.Println("Binary:")
				fmt.Printf("%v\n\n", hex.EncodeToString(contract.Body))
			}

			if shift {
				fmt.Println("Clause shift:")
				clauseMap, err := equityutil.Shift(contract)
				if err != nil {
					fmt.Println("Statistics contract clause shift error:", err)
					os.Exit(0)
				}

				for clause, shift := range clauseMap {
					fmt.Printf("    %s:  %v\n", clause, shift)
				}
				fmt.Printf("\nNOTE: \n    If the contract contains only one clause, Users don't need clause selector when unlock contract." +
					"\n    Furthermore, there is no signification for ending clause shift except for display.\n\n")
			}

			if instance {
				if i != len(contracts)-1 {
					continue
				}

				fmt.Println("Instantiated program:")
				if len(args)-1 < len(contract.Params) {
					fmt.Printf("Error: The number of input arguments %d is less than the number of contract parameters %d\n", len(args)-1, len(contract.Params))
					usage := fmt.Sprintf("Usage:\n  equity %s", args[0])
					for _, param := range contract.Params {
						usage = usage + " <" + param.Name + ">"
					}
					fmt.Printf("%s\n\n", usage)
					os.Exit(0)
				}

				contractArgs, err := equityutil.ConvertArguments(contract, args[1:len(contract.Params)+1])
				if err != nil {
					fmt.Println("Convert arguments into contract parameters error:", err)
					os.Exit(0)
				}

				instantProg, err := equityutil.InstantiateContract(contract, contractArgs)
				if err != nil {
					fmt.Println("Instantiate contract error:", err)
					os.Exit(0)
				}
				fmt.Printf("%v\n\n", hex.EncodeToString(instantProg))
			}
		}
	},
}
