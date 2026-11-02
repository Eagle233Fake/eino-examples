package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/schema"
	"github.com/joho/godotenv"
)

func main() {
	ctx := context.Background()
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}
	cm, err := agenticopenai.NewChatModel(ctx, &agenticopenai.ChatConfig{
		APIKey:  os.Getenv("API_KEY"),
		BaseURL: os.Getenv("BASE_URL"),
		Model:   os.Getenv("MODEL"),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	msg := []*schema.AgenticMessage{
		schema.SystemAgenticMessage("你是一个 Go 编程老师"),
		&schema.AgenticMessage{
			Role: schema.AgenticRoleTypeUser,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.UserInputText{Text: "什么是GMP"}),
			},
		},
	}

	// Using Generate
	//resp, err := cm.Generate(ctx, msg)
	//if err != nil {
	//	fmt.Println(err)
	//	os.Exit(1)
	//}
	//
	//for _, block := range resp.ContentBlocks {
	//	if block.AssistantGenText != nil {
	//		fmt.Println(block.AssistantGenText.Text)
	//	}
	//}

	// Using stream
	stream, err := cm.Stream(ctx, msg)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer stream.Close()
	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		for _, block := range msg.ContentBlocks {
			if block.AssistantGenText != nil {
				fmt.Print(block.AssistantGenText.Text)
			}
		}
	}

	//for _, msg := range msg {
	//	fmt.Println("Role: ", msg.Role)
	//	fmt.Println("Blocks: ", len(msg.ContentBlocks))
	//	fmt.Println("Type: ", msg.ContentBlocks[0].Type)
	//}
}
