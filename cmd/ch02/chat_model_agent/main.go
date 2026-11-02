package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

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

	events := runner.Query(ctx, "什么是GMP，一句话回答就好，简短一点")
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
					stream.Close()
					log.Fatal(err)
				}

				for _, block := range msg.ContentBlocks {
					if block.AssistantGenText != nil {
						fmt.Print(block.AssistantGenText.Text)
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
					fmt.Print(block.AssistantGenText.Text)
				}
			}
		}

		//fmt.Println("开始Msg：", time.Now().Format(time.RFC3339))
		//msg, err := event.Output.MessageOutput.GetMessage()
		//if err != nil {
		//	log.Fatal(err)
		//}
		//fmt.Println("结束Msg：", time.Now().Format(time.RFC3339))
		//
		//for _, block := range msg.ContentBlocks {
		//	if block.AssistantGenText != nil {
		//		fmt.Print(block.AssistantGenText.Text)
		//	}
		//}

		fmt.Println()
	}
}
