package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//msg := []*schema.AgenticMessage{
	//	schema.SystemAgenticMessage("你是一个 Go 编程老师"),
	//	schema.UserAgenticMessage("什么是GMP"),
	//}

	ctx := context.Background()
	cm, err := agenticopenai.NewChatModel(
		ctx,
		&agenticopenai.ChatConfig{
			APIKey:  os.Getenv("API_KEY"),
			BaseURL: os.Getenv("BASE_URL"),
			Model:   os.Getenv("MODEL"),
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewTypedChatModelAgent(
		ctx,
		&adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
			Name:        "GoTutor",
			Description: "Go 教师",
			Instruction: "你是一个 Go 编程老师",
			Model:       cm,
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewTypedRunner(
		adk.TypedRunnerConfig[*schema.AgenticMessage]{
			Agent:           agent,
			EnableStreaming: true,
		},
	)

	scanner := bufio.NewScanner(os.Stdin)
	history := []*schema.AgenticMessage{}
	for {
		fmt.Print("You: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}

		history = append(history, schema.UserAgenticMessage(input))
		events := runner.Run(ctx, history)

		var ans strings.Builder

		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				log.Fatal(event.Err)
			}
			if event.Output == nil || event.Output.MessageOutput == nil {
				continue
			}
			output := event.Output.MessageOutput
			if output.IsStreaming {
				stream := output.MessageStream
				for {
					msg, err := stream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						log.Fatal(err)
					}

					if msg == nil {
						continue
					}
					for _, block := range msg.ContentBlocks {
						if block.AssistantGenText != nil {
							text := block.AssistantGenText.Text
							fmt.Print(text)
							ans.WriteString(text)
						}
					}
				}
				stream.Close()
			} else {
				msg := output.Message
				if msg == nil {
					continue
				}
				for _, block := range msg.ContentBlocks {
					if block.AssistantGenText != nil {
						text := block.AssistantGenText.Text
						fmt.Print(text)
						ans.WriteString(text)
					}
				}
			}

		}
		history = append(history, &schema.AgenticMessage{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.AssistantGenText{Text: ans.String()}),
			},
		})

		fmt.Println()
	}
	if scanner.Err() != nil {
		log.Fatal(scanner.Err())
	}

}
