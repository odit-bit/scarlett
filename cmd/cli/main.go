package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/odit-bit/scarlett/api/cluster"
	"github.com/odit-bit/scarlett/api/rest"
	"github.com/spf13/cobra"
)

// var cli storeproto.StorerClient

// var gconn = func() *grpc.ClientConn {
// 	addr := "127.0.0.1:9090"

// }()

// func init() {
// 	cli = storeproto.NewStorerClient(gconn)
// }

// var client = Client{}
var cmd = cobra.Command{}

func init() {
	// cc, err := cluster.NewClient()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// client.cc = cc

	cmd.PersistentFlags().StringP("addr", "a", "", "")
	log.SetFlags(log.Lshortfile)
	cmd.AddCommand(&SetCmd, &GetCmd)
}

func main() {
	// client.clusterAddr = append(client.clusterAddr, "localhost:8080")
	cmd.Execute()
}

type Client struct {
	clusterAddr []string
	cc          *cluster.Client
}

type SetResponse struct {
	Result string
	Err    error
}

func (c *Client) Set(ctx context.Context, addrs, key, value string, v *SetResponse) error {
	result, err := c.cc.Set(ctx, addrs, key, value)
	if err != nil {
		return err
	}
	v.Result = result
	v.Err = err
	return nil

}

type GetResponse struct {
	Value string
	Error error
}

func (c *Client) Get(ctx context.Context, key string, v *GetResponse) error {
	m := rest.APIRoute
	r := m[rest.QUERY_API]
	method, path := r.Route()
	addr := c.clusterAddr[0]

	fullpath := filepath.Join(addr, path)
	endpoint := fmt.Sprintf("http://%s", fullpath)
	body := bytes.Buffer{}
	json.NewEncoder(&body).Encode(map[string]any{
		"cmd": "get",
		"key": key,
	})

	req, err := http.NewRequestWithContext(ctx, method, endpoint, &body)
	if err != nil {
		return err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(b, v); err != nil {
		return err
	}

	return nil

}

var SetCmd = cobra.Command{
	Use:        "set key value",
	Aliases:    []string{},
	SuggestFor: []string{},
	Short:      "",
	GroupID:    "",
	Long:       "",
	Example:    "set my-key my-value",
	Args:       cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("-addr flag", addr)

		client, err := cluster.NewClient()
		if err != nil {
			log.Println(err)
			return
		}
		defer client.Close()

		res, err := client.Set(cmd.Context(), addr, args[0], args[1])
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res)
		}


	},
}

var GetCmd = cobra.Command{
	Use:        "get key",
	Aliases:    []string{},
	SuggestFor: []string{},
	Short:      "",
	GroupID:    "",
	Long:       "",
	Example:    "get my-key",
	Args:       cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}

		client, err := cluster.NewClient()
		if err != nil {
			log.Println(err)
			return
		}
		defer client.Close()

		res, err := client.Get(cmd.Context(), addr, args[0])
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println("value:", string(res))
		}
	},
}
