// Copyright 2021 AccelByte Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"time"

	commonblobgo "github.com/AccelByte/common-blob-go"
	"github.com/AccelByte/iam-go-sdk"
	"github.com/go-redis/redis/v8"
	gormv1 "github.com/jinzhu/gorm"
	"github.com/olivere/elastic"
	"go.mongodb.org/mongo-driver/mongo"
	"gocloud.dev/gcerrors"
	"gorm.io/gorm"
)

var errClientNil = fmt.Errorf("client is nil")

// MongoHealthCheck is function for mongodb health check
func MongoHealthCheck(mongoClient *mongo.Client, timeout time.Duration) CheckFunc {
	return func() error {
		if mongoClient == nil {
			return errClientNil
		}

		ctxWithTimeout, ctxWithTimeoutCancel := context.WithTimeout(context.Background(), timeout)
		defer ctxWithTimeoutCancel()

		return mongoClient.Ping(ctxWithTimeout, nil)
	}
}

// IamHealthCheck is function for IAM health check
func IamHealthCheck(iamClient iam.Client) CheckFunc {
	return func() error {
		if iamClient == nil {
			return errClientNil
		}

		if !iamClient.HealthCheck() {
			return fmt.Errorf("IAM is unhealthy")
		}

		return nil
	}
}

// RedisHealthCheck is function for Redis health check
func RedisHealthCheck(redisClient *redis.Client, timeout time.Duration) CheckFunc {
	return func() error {
		if redisClient == nil {
			return errClientNil
		}

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		return redisClient.Ping(ctxWithTimeout).Err()
	}
}

// ElasticHealthCheck is function for Elastic health check
func ElasticHealthCheck(elasticClient *elastic.Client, host, port string, timeout time.Duration) CheckFunc {
	return func() error {
		if elasticClient == nil {
			return fmt.Errorf("unable to ping elastic search: client is nil")
		}

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)

		defer cancel()

		res, code, err := elasticClient.Ping(fmt.Sprintf("%s:%s", host, port)).Do(ctxWithTimeout)
		if err != nil {
			return fmt.Errorf("unable to ping elastic search: %s", err)
		}

		if code != http.StatusOK {
			return fmt.Errorf("unable to ping elastic search: expected status code = %d; got %d", http.StatusOK, code)
		}

		if res == nil {
			return fmt.Errorf("unable to ping elastic search: expected to return result, got: %v", res)
		}

		if res.Name == "" {
			return fmt.Errorf("unable to ping elastic search: expected Name != \"\"; got %q", res.Name)
		}

		if res.Version.Number == "" {
			return fmt.Errorf("unable to ping elastic search: expected Version.Number != \"\"; got %q", res.Version.Number)
		}

		return nil
	}
}

// PostgresHealthCheck is health check for Postgres with gorm V2 driver
func PostgresHealthCheck(postgreClient *gorm.DB, timeout time.Duration) CheckFunc {
	return func() error {
		if postgreClient == nil {
			return errClientNil
		}

		db, err := postgreClient.DB()
		if err != nil {
			return fmt.Errorf("unable to get postgres database: %v", err)
		}

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := db.PingContext(ctxWithTimeout); err != nil {
			return fmt.Errorf("unable to ping postgres database: %v", err)
		}

		return nil
	}
}

// PostgresHealthCheckV1 is health check for Postgres with gorm V1 driver
func PostgresHealthCheckV1(postgreClient *gormv1.DB, timeout time.Duration) CheckFunc {
	return func() error {
		if postgreClient == nil {
			return errClientNil
		}

		ctxWithTimeout, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := postgreClient.DB().PingContext(ctxWithTimeout); err != nil {
			return fmt.Errorf("unable to ping postgres database: %v", err)
		}

		return nil
	}
}

// CloudStorageCheck is function for check cloud straoge health based on AccelByte common-blob-go library
func CloudStorageCheck(cloudStorage commonblobgo.CloudStorage) CheckFunc {
	return func() error {

		if cloudStorage == nil {
			return errClientNil
		}

		// get attribute of random key, if error returns is other than error not found, meaning there's
		// an error at bucket provider service
		_, err := cloudStorage.Get(context.Background(), "randomKey")
		if gcerrors.Code(err) == gcerrors.NotFound || err == nil {
			return nil
		}
		return err
	}
}
