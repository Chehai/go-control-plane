package mysql

import (
	"fmt"
	"context"
	"encoding/json"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"

	log "github.com/sirupsen/logrus"


	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
)

const (
	dbType = "mysql"
	clustersTable = "clusters"
	endpointsTable = "endpoints"
	routesTable = "routes"
	createClustersTableSql = "CREATE TABLE IF NOT EXISTS " + \
												clustersTable + \
												"(id INT NOT NULL AUTO_INCREMENT, " + \
												"name VARCHAR(128) NOT NULL, " + \
												"conf VARCHAR(256) NOT NULL, " + \
												"created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, " + \
												"updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " + \
												"PRIMARY KEY(id), " + \
												"UNIQUE KEY(name))"
	createRoutesTableSql = "CREATE TABLE IF NOT EXISTS " + \
											routesTable + \
											"(id INT NOT NULL AUTO_INCREMENT, " + \
											"vhost VARCHAR(128) NOT NULL, " + \
											"cluster VARCHAR(128) NOT NULL, " + \
											"created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, " + \
											"updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP, " + \
											"PRIMARY KEY(id), " + \
											"UNIQUE KEY(vhost))"
	upsertClusterSql = "REPLACE INTO " + \
										 clustersTable + \
										 "(name, conf) VALUES (?, ?)"
	upsertRouteSql = "REPLACE INTO " + \
									 routesTable + \
									 "(vhost, cluster) VALUES (?, ?)"
	getRoutesSql = "SELECT vhost, cluster FROM " + routesTable
	getClustersSql = "SELECT conf FROM " + clustersTable
)

type Store struct {
	db *sql.DB
}

func (s *Store)Init(ctx context.Context, confFile string) error {
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
}

func (s *Store) UpsertCluster(ctx context.Context, cluster *common.Cluster) error {
	clusterJson, err := json.Marshal(*cluster)
	if err != nil {
		log.Error("Upserting cluster %v failed: %v", *cluster, err)
		return err
	}
	_, err := s.db.ExecContext(ctx, upsertClusterSql, cluster.Name, clusterJson)
	if err != nil {
		log.Error("Upserting cluster %v failed: %v", *cluster, err)
		return err
	}
	return nil
}

func (s *Store) UpsertRoute(ctx context.Context, route *common.Route) error {
	_, err := s.db.ExecContext(ctx, upsertRouteSql, route.Vhost, route.Cluster)
	if err != nil {
		log.Errorf("Upserting route %v failed: %v", *route, err)
		return err
	}
	return nil
}

func (s *Store) GetRoutes(ctx context.Context) ([]*common.Route, error) {
	ret := []*common.Route{}
	rows, err := s.db.QueryContext(ctx, getRoutesSql)
	if err != nil {
		log.Errorf("Getting routes failed: %v", err)
		return nil, err
	}
	defer rows.close()
	for rows.Next() {
		var route common.Route
		if err := rows.Scan(&route.Vhost, &route.Cluster); err != nil {
			log.Errorf("Geting routes failed: %v", err)
			return nil, err
		}
		ret = append(ret, &route)
	}
	if err = rows.Err(); err != nil {
		log.Errorf("Getting routes failed: %v", err)
		return nil, err
	}
	return ret, nil
}

func (s *Store) GetClusters(ctx context.Context) ([]*common.Cluster, error) {
	ret := []*common.Cluster{}
	rows, err := s.db.QueryContext(ctx, getClustersSql)
	if err != nil {
		log.Errorf("Getting clusters failed: %v", err)
		return nil, err
	}
	defer rows.close()
	for rows.Next() {
		var conf string
		if err := rows.Scan(&conf); err != nil {
			log.Errorf("Geting cluster failed: %v", err)
			return nil, err
		}
    cluster := common.Cluster{}
    if err := json.Unmarshal([]byte(conf), &cluster); err != nil {
			log.Errorf("Geting cluster failed: %v", err)
			return err
		}
		ret = append(ret, &cluster)
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
		log.Errorff("Connecting to %s failed: %v", dsn, err)
		return nil, err
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS ?", dbName)
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

	createTableSqls := []string{createClustersTableSql, createRoutesTableSql}
	for _, sql := range createTableSqls {
		_, err = db.ExecContext(ctx, sql)
		if err != nil {
			log.Errorf("Creating table %s failed: %v", sql, err)
			db.Close()
			return nil, err
		}
	}

	return db, nil
}
