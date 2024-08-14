package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type Result struct {
	IP    string   `json:"ip"`
	Hosts []string `json:"hostnames"`
}

func main() {
	hostToScan := os.Args[1]
	logs := false
	// if os.Args[2] != "" {
	// 	logs = true
	// }

	output, err := runDNSScan(hostToScan, logs)
	if err != nil {
		log.Println("Error running:", err)
		return
	}
	var scanner *bufio.Scanner
	scanner = bufio.NewScanner(strings.NewReader(output))
	hosts := make([]string, 0)
	for scanner.Scan() {
		hosts = append(hosts, scanner.Text())
	}
	final := host2ip(hosts)
	results := make([]Result, 0)
	for ip, hosts := range final {
		results = append(results, Result{IP: ip, Hosts: hosts})
	}
	jres, err := json.MarshalIndent(results, "", "    ")
	if err != nil {
		log.Println("Error marshalling:", err)
		return
	}
	filename := fmt.Sprintf("%s_ips.json", hostToScan)
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error creating file:", err)
	}
	defer file.Close()
	file.WriteString(string(jres))
	fmt.Println(string(jres))
}

func parseOutput(output string) string {
	o := strings.ReplaceAll(output, "Found: ", "")
	return o
}

func runDNSScan(tld string, showLogs bool) (string, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", err
	}
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:     "sherangee/dnsscan:latest",
		Tty:       true,
		OpenStdin: true,
		Env:       []string{fmt.Sprintf("TLD=%s", tld)},
	}, &container.HostConfig{
		Binds:      []string{fmt.Sprintf("%s:/out", currentDir)},
		AutoRemove: true,
	}, nil, nil, tld)
	if err != nil {
		return "", err
	}
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", err
	}

	// Create a channel to receive OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	// Create a channel to receive when the container stops
	doneCh := make(chan error, 1)

	go func() {
		statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
		select {
		case err := <-errCh:
			if err != nil {
				doneCh <- err
			}
		case <-statusCh:
			doneCh <- nil
		}
	}()

	if showLogs {
		go func() {
			out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true, Follow: true})
			if err != nil {
				fmt.Println(err.Error())
			}
			defer out.Close()
			buf := make([]byte, 4096)
			for {
				n, err := out.Read(buf)
				if err == io.EOF {
					break
				}
				if err != nil {
					fmt.Println(err.Error())
					break
				}
				fmt.Print(string(buf[:n]))
			}
		}()
	}

	timeout := 5

	// Select on either a signal or the done channel
	select {
	case err := <-doneCh:
		if err != nil {
			return "", err
		}
	case <-sigCh:
		// If a signal is received, stop the container
		if err := cli.ContainerStop(ctx, resp.ID, container.StopOptions{
			Timeout: &timeout,
		}); err != nil {
			return "", err
		}
	}

	filename := fmt.Sprintf("%s.hosts.txt", tld)
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()
	output, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	final := parseOutput(string(output))

	return final, nil
}
