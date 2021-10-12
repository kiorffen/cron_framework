package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	es "cron_framework/common/elastic"
	mc "cron_framework/common/mysql"
	rc "cron_framework/common/redis"

	log "cron_framework/common/logger"

	"cron_framework/dict"
	"cron_framework/handler"
	"cron_framework/tools"
	"cron_framework/worker"

	"go.uber.org/zap"
)

const (
	INTERVAL_SECOND = 86400
	LIMIT_COUNT     = 10000

	CONF_FILE = "./conf/main.conf"
)

var (
	G_globalExit bool

	G_env string

	G_interval int

	G_svrPort string

	G_isRunning bool

	G_logger *log.Logger

	G_conf tools.GlobalConf

	G_mc map[string]*mc.Mysql
	G_es map[string]*es.ElasticClient
	G_rc map[string]*rc.Redis

	G_testDict dict.TestDict

	G_esIndex string
	G_esType  string
)

func init() {
	G_globalExit = false

	G_env = "dev"
	G_interval = 0
	G_svrPort = "80"
	G_isRunning = false

	G_mc = make(map[string]*mc.Mysql)
	G_es = make(map[string]*es.ElasticClient)
	G_rc = make(map[string]*rc.Redis)
}

func initLogger() error {
	var err error
	conf := log.LoggerConf{
		FilePath:    G_conf.Log.FilePath,
		IsLocalTime: true,
		MaxSize:     1024, // 1G
		MaxBackups:  30,   // 10*1G
		MaxDays:     G_conf.Log.MaxDays,
		IsCompress:  false,
		Level:       G_conf.Log.Level,
		ServerName:  G_conf.ServerName,
	}

	G_logger, err = log.NewLogger(conf)
	if err != nil {
		return err
	}

	return nil
}

func globalInit() error {
	var err error

	err = initMysql()
	if err != nil {
		G_logger.Logger().Error("init mysql failed", zap.String("errmsg", err.Error()))
		return err
	}

	err = initElastic()
	if err != nil {
		G_logger.Logger().Error("init es failed", zap.String("errmsg", err.Error()))
		return err
	}

	err = initRedis()
	if err != nil {
		G_logger.Logger().Error("init redis failed", zap.String("errmsg", err.Error()))
		return err
	}

	G_testDict.AutoReload(G_env)

	if G_env == "release" {
		G_esIndex = "yxs_gicp_index"
		G_esType = "contents"
	} else {
		G_esIndex = "yxs_gicp_index_test"
		G_esType = "contents"
	}

	return nil
}

func initMysql() error {
	for _, item := range G_conf.Mysql {
		mysqlConf := mc.MysqlConf{
			Address:      item.Addr,
			Timeout:      time.Duration(item.Timeout) * time.Second,
			MaxIdleConns: item.MaxIdle,
			MaxOpenConns: item.MaxOpen,
		}

		m := mc.New(mysqlConf)
		if m == nil {
			return errors.New("create mysql client failed.")
		}

		G_mc[item.Name] = m
	}

	return nil
}

func initElastic() error {
	for _, item := range G_conf.Elastic {
		esConf := es.ElasticConf{
			Address:  item.Addr,
			MaxRetry: item.MaxRetry,
			User:     item.User,
			Password: item.Password,
		}

		e, err := es.New(esConf)
		if err != nil {
			return err
		}

		G_es[item.Name] = e
	}

	return nil
}

func initRedis() error {
	for _, item := range G_conf.Redis {
		redisConf := rc.RedisConf{
			Address:   item.Addr,
			Timeout:   time.Duration(item.Timeout) * time.Second,
			Password:  item.Password,
			MaxIdle:   item.MaxIdle,
			MaxActive: item.MaxActive,
		}

		r := rc.New(redisConf)
		if r == nil {
			return errors.New("create redis failed.")
		}

		G_rc[item.Name] = r
	}

	return nil
}

func process() {
	if G_env == "pre" {
		G_logger.Logger().Info("pre env do not process")
		return
	}

	if G_isRunning == true {
		G_logger.Logger().Info("last process is running")
		return
	}
	G_isRunning = true

	// work step
	worker.TestWorker()

	G_isRunning = false
}

func main() {
	var err error

	go signalProcess()

	err = G_conf.ParseConf(CONF_FILE)
	if err != nil {
		fmt.Println("parse conf failed. err:" + err.Error())
	}

	err = initLogger()
	if err != nil {
		fmt.Println("initLogger failed. err: " + err.Error())
	}

	G_interval = G_conf.Interval
	G_env = G_conf.Env
	G_svrPort = G_conf.ServerPort
	G_logger.Logger().Info("",
		zap.String("conf_env", G_env),
		zap.Int("interval", G_interval),
		zap.String("server_port", G_svrPort))

	err = globalInit()
	if err != nil {
		G_logger.Logger().Fatal("global init failed", zap.String("errmsg", err.Error()))
		return
	}

	go StartWebServer()

	for {
		stimeProc := time.Now().Format("2006-01-02 15:04:05")
		if G_globalExit {
			G_logger.Logger().Info("server exist")
			break
		}
		process()
		etimeProc := time.Now().Format("2006-01-02 15:04:05")
		G_logger.Logger().Info("process time",
			zap.String("stime", stimeProc),
			zap.String("etime", etimeProc))
		time.Sleep(INTERVAL_SECOND * time.Second)
	}
}

func StartWebServer() {
	http.HandleFunc("/process", handler.TestHandler)

	err := http.ListenAndServe(":"+G_svrPort, nil)
	if err != nil {
		fmt.Println("ListenAndServe failed. err:%s\n", err.Error())
	}
}

func signalProcess() {
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	for s := range c {
		switch s {
		case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
			fmt.Println("exit signal")
			exitFunc()
		default:
			fmt.Println("other signal")
		}
	}
}

func exitFunc() {
	fmt.Println("pre exit")
	G_globalExit = true
	// http serve chan没有办法接收G_globalExit，等待一个最大超时时间退出，这里设置3秒
	time.Sleep(time.Duration(G_conf.ServerWaitTimeout) * time.Millisecond)
	fmt.Println("clear")
	fmt.Println("finish exit")

	os.Exit(0)
}
