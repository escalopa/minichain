// Package cli defines the cobra command tree. Commands stay thin:
// load keys via keystore, talk to the node via nodeclient, print.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/escalopa/minichain/wallet/internal/keystore"
	"github.com/escalopa/minichain/wallet/internal/nodeclient"
	"github.com/escalopa/minichain/wallet/internal/tx"
)

func New() *cobra.Command {
	var nodeURL, walletFile string

	root := &cobra.Command{
		Use:           "wallet",
		Short:         "minichain CLI wallet: keys, balances and transfers",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&nodeURL, "node", envOr("NODE_URL", "http://localhost:3000"), "node base URL")
	root.PersistentFlags().StringVar(&walletFile, "file", envOr("WALLET_FILE", defaultWalletPath()), "wallet file path")

	// Constructed lazily inside closures, not up front: persistent
	// flags are only parsed when a command actually runs, so building
	// the client here would capture the default URL. Loading the
	// wallet lazily also means `--help` works without a key file.
	client := func() *nodeclient.Client { return nodeclient.New(nodeURL) }
	wallet := func() (*keystore.Wallet, error) { return keystore.Load(walletFile) }

	var force bool
	keygen := &cobra.Command{
		Use:   "keygen",
		Short: "generate a key pair and save it to the wallet file",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Overwriting a wallet destroys the only copy of the seed
			// and therefore the coins. Refusing by default is the
			// single most important safety check in this binary.
			if _, err := os.Stat(walletFile); err == nil && !force {
				return fmt.Errorf("%s already exists; use --force to overwrite", walletFile)
			}
			w, err := keystore.Generate()
			if err != nil {
				return err
			}
			if err := w.Save(walletFile); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wallet saved to %s\naddress: %s\n", walletFile, w.Address())
			return nil
		},
	}
	keygen.Flags().BoolVar(&force, "force", false, "overwrite an existing wallet file")

	address := &cobra.Command{
		Use:   "address",
		Short: "print the wallet address",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := wallet()
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), w.Address())
			return nil
		},
	}

	balance := &cobra.Command{
		Use:   "balance [address]",
		Short: "show the balance of an address (defaults to your own)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			addr, err := addressArg(args, wallet)
			if err != nil {
				return err
			}
			amount, err := client().Balance(cmd.Context(), addr)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s\nbalance: %d\n", addr, amount)
			return nil
		},
	}

	var to string
	var amount uint64
	send := &cobra.Command{
		Use:   "send",
		Short: "sign a transfer locally and submit it to the node",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := wallet()
			if err != nil {
				return err
			}
			// The three steps that make this a real wallet:
			//   1. ask the node which nonce it expects next,
			//   2. build and sign the transaction locally,
			//   3. hand over only the signed result.
			// The private key never crosses the process boundary, so
			// the node can be someone else's machine entirely.
			c := client()
			nonce, err := c.Nonce(cmd.Context(), w.Address())
			if err != nil {
				return err
			}
			t := &tx.Transaction{
				From:      w.Address(),
				To:        to,
				Amount:    amount,
				Nonce:     nonce,
				Timestamp: uint64(time.Now().UnixMilli()),
			}
			// Sign last: Payload() reads every other field, so the
			// signature must be computed once they are all final.
			t.Signature = w.Sign(t.Payload())
			if err := c.Submit(cmd.Context(), t); err != nil {
				return err
			}
			// OutOrStdout, not cmd.Printf: cobra's Print* helpers write
			// to stderr, which silently broke `ADDR=$(wallet address)`
			// in shell pipelines until an end-to-end run caught it.
			fmt.Fprintf(cmd.OutOrStdout(), "accepted into mempool: %d -> %s (nonce %d)\n", amount, short(to), nonce)
			fmt.Fprintln(cmd.OutOrStdout(), "run `wallet mine` to include it in a block")
			return nil
		},
	}
	send.Flags().StringVar(&to, "to", "", "recipient address")
	send.Flags().Uint64Var(&amount, "amount", 0, "amount to transfer")
	send.MarkFlagRequired("to")
	send.MarkFlagRequired("amount")

	mine := &cobra.Command{
		Use:   "mine",
		Short: "mine the mempool, taking the block reward yourself",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := wallet()
			if err != nil {
				return err
			}
			index, hash, err := client().Mine(cmd.Context(), w.Address())
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "mined block #%d %s\n", index, hash)
			return nil
		},
	}

	root.AddCommand(keygen, address, balance, send, mine)
	return root
}

func addressArg(args []string, wallet func() (*keystore.Wallet, error)) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	w, err := wallet()
	if err != nil {
		return "", err
	}
	return w.Address(), nil
}

func defaultWalletPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "wallet.json"
	}
	return filepath.Join(home, ".minichain", "wallet.json")
}

func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}

func short(s string) string {
	if len(s) <= 16 {
		return s
	}
	return s[:8] + "…" + s[len(s)-8:]
}
