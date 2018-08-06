package mysql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	_ "github.com/go-sql-driver/mysql"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
)

const (
	dbType                 = "mysql"
	clustersTable          = "clusters"
	endpointsTable         = "endpoints"
	routesTable            = "routes"
	createClustersTableSQL = "CREATE TABLE IF NOT EXISTS " +
		clustersTable +
		"(id INT NOT NULL AUTO_INCREMENT, " +
		"name VARCHAR(128) NOT NULL, " +
		"conf VARCHAR(256) NOT NULL, " +
		"created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, " +
		"updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " +
		"PRIMARY KEY(id), " +
		"UNIQUE KEY(name))"
	createRoutesTableSQL = "CREATE TABLE IF NOT EXISTS " +
		routesTable +
		"(id INT NOT NULL AUTO_INCREMENT, " +
		"vhost VARCHAR(128) NOT NULL, " +
		"cluster VARCHAR(128) NOT NULL, " +
		"created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, " +
		"updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " +
		"PRIMARY KEY(id), " +
		"UNIQUE KEY(vhost))"
	upsertClusterSQL = "REPLACE INTO " + clustersTable + "(name, conf) VALUES (?, ?)"
	deleteClusterSQL = "DELETE FROM " + clustersTable + " WHERE name = ?"
	getClustersSQL   = "SELECT conf FROM " + clustersTable
	upsertRouteSQL   = "REPLACE INTO " + routesTable + "(vhost, cluster) VALUES (?, ?)"
	deleteRouteSQL   = "DELETE FROM " + routesTable + " WHERE vhost = ?"
	getRouteSQL      = "SELECT vhost, cluster FROM " + routesTable + " WHERE vhost = ? LIMIT 1"
	getRoutesSQL     = "SELECT vhost, cluster FROM " + routesTable
)

type Store struct {
	db *sql.DB
}

func (s *Store) Init(ctx context.Context, confFile string) error {
	// read db config from file
	err := createDb(ctx, "root", "", "127.0.0.1", 3306, "compass")
	if err != nil {
		log.Error("Creating db failed")
		return err
	}

	s.db, err = initDb(ctx, "root", "", "127.0.0.1", 3306, "compass")
	if err != nil {
		log.Error("Initing db failed")
		return err
	}
	return nil
}

func (s *Store) UpsertCluster(ctx context.Context, cluster *common.Cluster) error {
	clusterJSON, err := json.Marshal(*cluster)
	if err != nil {
		log.Error("Upserting cluster %v failed: %v", *cluster, err)
		return err
	}
	_, err = s.db.ExecContext(ctx, upsertClusterSQL, cluster.Name, clusterJSON)
	if err != nil {
		log.Error("Upserting cluster %v failed: %v", *cluster, err)
		return err
	}
	return nil
}

func (s *Store) DeleteCluster(ctx context.Context, clusterName string) error {
	_, err := s.db.ExecContext(ctx, deleteClusterSQL, clusterName)
	if err != nil {
		log.Error("Deleting cluster %s failed: %v", clusterName, err)
		return err
	}
	return nil
}

func (s *Store) UpsertRoute(ctx context.Context, route *common.Route) error {
	_, err := s.db.ExecContext(ctx, upsertRouteSQL, route.Vhost, route.Cluster)
	if err != nil {
		log.Errorf("Upserting route %v failed: %v", *route, err)
		return err
	}
	return nil
}

func (s *Store) DeleteRoute(ctx context.Context, vhost string) error {
	_, err := s.db.ExecContext(ctx, deleteRouteSQL, vhost)
	if err != nil {
		log.Errorf("Deleting route %s failed: %v", vhost, err)
		return err
	}
	return nil
}

func (s *Store) GetRoute(ctx context.Context, vhost string) (*common.Route, error) {
	ret := common.Route{}
	err := s.db.QueryRowContext(ctx, getRouteSQL, vhost).Scan(&ret.Vhost, &ret.Cluster)
	if err != nil {
		log.Errorf("Getting route %s failed: %v", vhost, err)
		return nil, err
	}
	return &ret, nil
}

func (s *Store) GetRoutes(ctx context.Context) ([]*common.Route, error) {
	ret := []*common.Route{}
	rows, err := s.db.QueryContext(ctx, getRoutesSQL)
	if err != nil {
		log.Errorf("Getting routes failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		route := new(common.Route)
		if err := rows.Scan(&route.Vhost, &route.Cluster); err != nil {
			log.Errorf("Geting routes failed: %v", err)
			return nil, err
		}
		ret = append(ret, route)
	}
	if err = rows.Err(); err != nil {
		log.Errorf("Getting routes failed: %v", err)
		return nil, err
	}
	return ret, nil
}

func (s *Store) GetClusters(ctx context.Context) ([]*common.Cluster, error) {
	ret := []*common.Cluster{}
	rows, err := s.db.QueryContext(ctx, getClustersSQL)
	if err != nil {
		log.Errorf("Getting clusters failed: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var conf string
		if err := rows.Scan(&conf); err != nil {
			log.Errorf("Geting cluster failed: %v", err)
			return nil, err
		}
		cluster := new(common.Cluster)
		if err := json.Unmarshal([]byte(conf), cluster); err != nil {
			log.Errorf("Geting cluster failed: %v", err)
			return nil, err
		}
		ret = append(ret, cluster)
	}
	if err = rows.Err(); err != nil {
		log.Errorf("Getting clusters failed: %v", err)
		return nil, err
	}
	return ret, nil
}

func createDb(ctx context.Context, user string, password string, host string, port uint32, dbName string) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	db, err := sql.Open(dbType, dsn)
	if err != nil {
		log.Errorf("Connecting to %s failed: %v", dsn, err)
		return err
	}
	defer db.Close()

	// We assume dbName is from a trusted source like a configuration file, so no SQL injection here.
	// golang's database/sql package does not support ? placeholder in CREATE TABLE statements.
	// golang's database/sql does not expose any quoting method either.
	// https://github.com/golang/go/issues/18478
	_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+dbName)
	if err != nil {
		log.Errorf("Creating database %s failed: %v", dbName, err)
		return err
	}

	return nil
}

func initDb(ctx context.Context, user string, password string, host string, port uint32, dbName string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbName)
	db, err := sql.Open(dbType, dsn)
	if err != nil {
		log.Errorf("Connecting to %s failed: %v", dsn, err)
		return nil, err
	}

	createTableSQLs := []string{createClustersTableSQL, createRoutesTableSQL}
	for _, sql := range createTableSQLs {
		_, err = db.ExecContext(ctx, sql)
		if err != nil {
			log.Errorf("Creating table %s failed: %v", sql, err)
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
