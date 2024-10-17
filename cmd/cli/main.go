package main

import (
	"fmt"
	"log"

	storepb "github.com/odit-bit/scarlett/store/storepb"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var cmd = cobra.Command{}

func init() {
	cmd.PersistentFlags().StringP("addr", "a", "", "")
	log.SetFlags(log.Lshortfile)
	cmd.AddCommand(&SetCmd, &GetCmd, &GetLeaderCmd, &JoinCMD, &RemoveNodeCMD)
}

func main() {
	// client.clusterAddr = append(client.clusterAddr, "localhost:8080")
	cmd.Execute()
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
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		client := storepb.NewCommandClient(conn)
		res, err := client.Set(cmd.Context(), &storepb.SetRequest{
			Cmd:   storepb.Command_Type_Set,
			Key:   []byte(args[0]),
			Value: []byte(args[1]),
		})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res)
		}

	},
}

var GetCmd = cobra.Command{
	Use:     "get key",
	Example: "get my-key",
	Args:    cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}

		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatal(err)
		}
		defer conn.Close()

		client := storepb.NewCommandClient(conn)
		if err != nil {
			log.Println(err)
			return
		}

		res, err := client.Get(cmd.Context(), &storepb.GetRequest{
			Cmd: storepb.QueryType_Get,
			Key: []byte(args[0]),
		})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Printf("value: %s \n", string(res.Value))
		}
	},
}

var GetLeaderCmd = cobra.Command{
	Use:     "leader",
	Example: "leader",
	Args:    cobra.ExactArgs(0),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		client := storepb.NewClusterClient(conn)
		res, err := client.Leader(cmd.Context(), &storepb.LeaderRequest{})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res.Id, res.Addr)
		}

	},
}

var JoinCMD = cobra.Command{
	Use:        "join id address",
	Aliases:    []string{},
	SuggestFor: []string{},
	Short:      "",
	GroupID:    "",
	Long:       "",
	Example:    "join node-2 localhost:8002",
	Args:       cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		client := storepb.NewClusterClient(conn)
		res, err := client.Join(cmd.Context(), &storepb.JoinRequest{
			Address: args[1],
			Id:      args[0],
		})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res.Msg)
		}

	},
}

var RemoveNodeCMD = cobra.Command{
	Use:        "rm id",
	Aliases:    []string{},
	SuggestFor: []string{},
	Short:      "",
	GroupID:    "",
	Long:       "",
	Example:    "rm node-2",
	Args:       cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		addr, err := cmd.Flags().GetString("addr")
		if err != nil {
			log.Println(err)
			return
		}
		conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Println(err)
			return
		}
		defer conn.Close()

		client := storepb.NewClusterClient(conn)
		res, err := client.RemoveNode(cmd.Context(), &storepb.RemoveRequest{
			Id: args[0],
		})
		if err != nil {
			log.Fatal(err)
		} else {
			fmt.Println(res.Msg)
		}

	},
}
