package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// APIClient 简易 HTTP 客u6237端，连接 daemon 本地端口
// u6682时保存最后一次 API 地址
var defaultAPIAddr = "http://127.0.0.1:0" // 后续自发现

func main() {
	var rootCmd = &cobra.Command{
		Use:   "bitchat chat",
		Short: "FARP CLI chat client",
	}

	rootCmd.PersistentFlags().String("api-addr", "", "daemon local API address (e.g. http://127.0.0.1:49281)")

	rootCmd.AddCommand(
		statusCmd(),
		contactsCmd(),
		msgCmd(),
		creditCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show node status from daemon",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := mustAddr(cmd)
			resp, err := getJSON(addr + "/status")
			if err != nil {
				return err
			}
			fmt.Println(prettyJSON(resp))
			return nil
		},
	}
}

func contactsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "contacts",
		Short: "List contacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := mustAddr(cmd)
			resp, err := getJSON(addr + "/contacts")
			if err != nil {
				return err
			}
			fmt.Println(prettyJSON(resp))
			return nil
		},
	}
}

func creditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "credits",
		Short: "Show credit balance and status",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := mustAddr(cmd)
			resp, err := getJSON(addr + "/credits")
			if err != nil {
				return err
			}
			fmt.Println(prettyJSON(resp))
			return nil
		},
	}
}

func msgCmd() *cobra.Command {
	var to, text string
	c := &cobra.Command{
		Use:   "msg",
		Short: "Send or list messages (placeholder)",
		RunE: func(cmd *cobra.Command, args []string) error {
			addr := mustAddr(cmd)
			if to == "" {
				// 列出消息
				resp, err := getJSON(addr + "/messages/placeholder")
				if err != nil {
					return err
				}
				fmt.Println(prettyJSON(resp))
				return nil
			}
			// 发送消息: 调用 daemon /outbox
			_ = text
			fmt.Printf("🚀 not yet implemented: send to %s\n", to)
			return nil
		},
	}
	c.Flags().StringVar(&to, "to", "", "recipient FARP-ID")
	c.Flags().StringVar(&text, "text", "", "message text")
	return c
}

func mustAddr(cmd *cobra.Command) string {
	// 先尝u552f嗨发现（后续可以从 daemon 日志或运行状态文件中读取）
	addr := cmd.Flag("api-addr").Value.String()
	if addr != "" {
		return addr
	}
	fmt.Fprintln(os.Stderr, "error: daemon not running or --api-addr not set")
	os.Exit(1)
	return ""
}

func getJSON(url string) (map[string]any, error) {
	// 简易 HTTP GET
	// TODO: 完善
	return nil, fmt.Errorf("placeholder: would fetch %s", url)
}

func prettyJSON(v map[string]any) string {
	b, _ := json.MarshalIndent(v, "", "  ")
	return string(b)
}
