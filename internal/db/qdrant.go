package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/qdrant/go-client/qdrant"
	"github.com/sshfz/consumer-service-substrate/cmd/app/mq"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var client *qdrant.Client

func SaveEmbedding(key string, code string, imports []float32, importsDesc string, value []float32) {
	collectionName := "sleepyhead_slyme"
	client = mq.GetQDrantClient()
	ctx := context.Background()

	//check if the collection exists -----
	coll, err := client.CollectionExists(ctx, collectionName)

	if err != nil {
		log.Fatalf("Failed to list collections: %v", err)
	}

	if !coll {
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     1536,
				Distance: qdrant.Distance_Cosine,
			}),
		})

		if err != nil {
			log.Fatalln(err)
		}
	} else {
		log.Println("Collections already exists, skipping collection creation.")
	}
	// ///

	_, err = client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: collectionName,
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{
					PointIdOptions: &qdrant.PointId_Num{
						Num: uint64(time.Now().UnixNano()), // Better ID generation
					},
				},
				Vectors: &qdrant.Vectors{
					VectorsOptions: &qdrant.Vectors_Vector{
						Vector: &qdrant.Vector{
							Data: imports, // Make sure this is []float32
						},
					},
				},
				Payload: map[string]*qdrant.Value{
					"file": {
						Kind: &qdrant.Value_StringValue{
							StringValue: key,
						},
					},
					"code": {
						Kind: &qdrant.Value_StringValue{
							StringValue: code,
						},
					},
					"imports": {
						Kind: &qdrant.Value_StringValue{
							StringValue: importsDesc,
						},
					},
				},
			},
		},
	})

	if err != nil {
		log.Printf("Failed to upsert points: %v", err)
	}
}

func getCLient() qdrant.PointsClient {
	conn, err := grpc.Dial("localhost:6334", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect to qdrant: %v", err)
	}
	defer conn.Close()

	client := qdrant.NewPointsClient(conn)
	return client
}

func Search(queryVector []float32) {
	conn, err := grpc.Dial("localhost:6334", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to Qdrant: %v", err)
	}
	ctx := context.Background()

	pointsClient := qdrant.NewPointsClient(conn)

	res, err := pointsClient.Search(ctx, &qdrant.SearchPoints{
		CollectionName: "sleepyhead_slyme",
		Vector:         queryVector,
		Limit:          10,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{
				Enable: true,
			},
		},
	})
	if err != nil {
		log.Fatalf("Search failed: %v", err)
	}

	for _, point := range res.Result {
		// fmt.Println("ID:", point.Id)
		// fmt.Println("Payload:", point.Payload)
		fmt.Println("Score:", point.Score)
		// fmt.Println("-----")
		if file, ok := point.Payload["file"]; ok {
			fmt.Println("Fileeeeeeee:", file.GetStringValue())
		}
	}
}
