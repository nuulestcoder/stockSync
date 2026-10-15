package mongoclient

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Client manages the MongoDB client and the application database handle.
type Client struct {
	client   *mongo.Client
	database *mongo.Database
}

// New connects to MongoDB, verifies the connection, and selects the database.
func New(ctx context.Context, uri, databaseName string) (*Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongoclient: MongoDB URI is required")
	}
	if databaseName == "" {
		return nil, fmt.Errorf("mongoclient: database name is required")
	}

	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongoclient: connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		disconnectCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Disconnect(disconnectCtx)
		return nil, fmt.Errorf("mongoclient: ping: %w", err)
	}

	return &Client{
		client:   client,
		database: client.Database(databaseName),
	}, nil
}

// Database returns the configured application database.
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Client returns the underlying MongoDB client for transactions and advanced operations.
func (c *Client) Client() *mongo.Client {
	return c.client
}

// Ping verifies that MongoDB is reachable.
func (c *Client) Ping(ctx context.Context) error {
	if err := c.client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("mongoclient: ping: %w", err)
	}
	return nil
}

// Disconnect closes the MongoDB connection.
func (c *Client) Disconnect(ctx context.Context) error {
	if err := c.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongoclient: disconnect: %w", err)
	}
	return nil
}
