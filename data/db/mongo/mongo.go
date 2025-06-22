// Package db provides database utilities for MongoDB connection and ID generation
package mongo

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// New creates a new MongoDB connection with the given connection string and database name.
// It also returns a cleanup function to close the connection.
func New(ctx context.Context, dbsn, dbname string) (*mongo.Database, func(), error) {
	if dbsn == "" {
		return nil, nil, errors.New("mongo dbsn is empty")
	}
	if dbname == "" {
		return nil, nil, errors.New("mongo dbname is empty")
	}

	opts := options.Client().
		ApplyURI(fmt.Sprintf("mongodb://%s", dbsn)).
		SetWriteConcern(writeconcern.Majority()).
		SetRetryWrites(false).
		SetReadPreference(readpref.SecondaryPreferred())

	cli, err := mongo.Connect(opts)
	if err != nil {
		return nil, nil, errors.Wrap(err, "connect to mongo failed")
	}

	cleanup := func() {
		if disconnectErr := cli.Disconnect(context.Background()); disconnectErr != nil {
			slog.Error("mongo disconnect failed", "error", disconnectErr)
		}
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err = cli.Ping(pingCtx, readpref.Primary()); err != nil {
		cleanup()
		return nil, nil, errors.Wrap(err, "mongo ping failed")
	}

	db := cli.Database(dbname)

	return db, cleanup, nil
}
