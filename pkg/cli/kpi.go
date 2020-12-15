// SPDX-FileCopyrightText: 2020-present Open Networking Foundation <info@opennetworking.org>
//
// SPDX-License-Identifier: LicenseRef-ONF-Member-1.0

package cli

import (
	"context"
	"fmt"
	"text/tabwriter"
	"github.com/onosproject/onos-lib-go/pkg/cli"
	"github.com/spf13/cobra"
	kpimonapi "github.com/onosproject/onos-api/go/onos/kpimon"
)

func getListNumActiveUEsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "numues",
		Short: "Get the number of active UEs",
		RunE:  runListNumActiveUEsCommand,
	}
	cmd.Flags().Bool("no-headers", false, "disables output headers")
	return cmd
}

func runListNumActiveUEsCommand(cmd *cobra.Command, args []string) error {
	noHeaders, _ := cmd.Flags().GetBool("no-headers")
	conn, err := cli.GetConnection(cmd)
	if err != nil {
		return err
	}
	defer conn.Close()
	outputWriter := cli.GetOutput()
	writer := new(tabwriter.Writer)
	writer.Init(outputWriter, 0, 0, 3, ' ', tabwriter.FilterHTML)

	if !noHeaders {
		_, _ = fmt.Fprintln(writer, "Key[PLMNID, nodeID]\tnum(Active UEs)")
	}

	request := kpimonapi.GetRequest{
		Id: "kpimon",
	}

	client := kpimonapi.NewKpimonClient(conn)

	response, err := client.GetNumActiveUEs(context.Background(), &request)

	if err != nil {
		return err
	}

	for k, v := range response.GetObject().GetAttributes() {
		_, _ = fmt.Fprintf(writer, "%s\t%v\n", k, v)
	}

	_ = writer.Flush()

	return nil
}