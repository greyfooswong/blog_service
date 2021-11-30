package main

import (
	"blog-service/global"
	"blog-service/internal/model"
	"blog-service/internal/routers"
	"blog-service/pkg"
	"blog-service/pkg/logger"
	setting2 "blog-service/pkg/setting"
	"context"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	port    string
	runMode string
	config  string

	isVersion    bool
	buildTime    string
	buildVersion string
	gitCommitID  string
)

// @title 博客系统
// @version 1.0
// @description blog_service
// @termsOfService southli
func init() {
	err := setupSetting()
	if err != nil {
		log.Fatalf("init.setupSetting err : %v", err)
	}
	err = setupDBEngine()
	if err != nil {
		log.Fatalf("init.setupDBEngine err : %v", err)
	}
	err = setupLogger()
	if err != nil {
		log.Fatalf("init.setupLogger err : %v", err)
	}
	err = setupTracer()
	if err != nil {
		log.Fatalf("init.setupTrace err : %v", err)
	}
	setupFlag()
}

func main() {
	if isVersion {
		fmt.Printf("build_time : %s \n", buildTime)
		fmt.Printf("build_version: %s \n", buildVersion)
		fmt.Printf("git_commit_id: %s \n", gitCommitID)
		return
	}
	gin.SetMode(global.ServerSetting.RunMode)
	router := routers.NewRouter()
	s := &http.Server{
		Addr:           ":" + global.ServerSetting.HttpPort,
		Handler:        router,
		ReadTimeout:    global.ServerSetting.ReadTimeout,
		WriteTimeout:   global.ServerSetting.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("s.ListenAndServe err: %v", err)
		}
	}()
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shuting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatal("Server forced tp shutdown:", err)
	}
	log.Println("Server exiting")
	s.ListenAndServe()

}

func setupSetting() error {
	setting, err := setting2.NewSetting(strings.Split(config, ",")...)
	if err != nil {
		return err
	}
	err = setting.ReadSection("Server", &global.ServerSetting)
	if err != nil {
		return err
	}
	err = setting.ReadSection("JWT", &global.JWTSetting)
	if err != nil {
		return err
	}
	global.JWTSetting.Expire *= time.Second
	err = setting.ReadSection("App", &global.AppSetting)
	if err != nil {
		return err
	}
	err = setting.ReadSection("Database", &global.DatabaseSetting)
	if err != nil {
		return err
	}
	err = setting.ReadSection("Email", &global.EmailSetting)
	if err != nil {
		return err
	}
	if port != "" {
		global.ServerSetting.HttpPort = port
	}
	if runMode != "" {
		global.ServerSetting.RunMode = runMode
	}

	global.ServerSetting.ReadTimeout *= time.Second
	global.ServerSetting.WriteTimeout *= time.Second
	return nil
}

func setupDBEngine() error {
	var err error
	global.DBEngine, err = model.NewDBEngine(global.DatabaseSetting)
	if err != nil {
		return err
	}
	return nil
}

func setupLogger() error {
	global.Logger = logger.NewLogger(
		&lumberjack.Logger{
			Filename:  global.AppSetting.LogSavePath + "/" + global.AppSetting.LogFileName + global.AppSetting.LogFileExt,
			MaxSize:   600,
			MaxAge:    10,
			LocalTime: true,
		}, "", log.LstdFlags).WithCaller(2)
	return nil
}

func setupTracer() error {
	jaegerTracer, _, err := pkg.NewJaegerTracer("blog-service", "1.117.46.16:6831")
	if err != nil {
		return err
	}
	global.Tracer = jaegerTracer
	return nil
}

func setupFlag() error {
	flag.StringVar(&port, "port", "", "启动端口")
	flag.StringVar(&runMode, "mode", "", "启动模式")
	flag.StringVar(&config, "config", "configs/", "指定要使用的配置文件路径")
	flag.BoolVar(&isVersion, "version", false, "编译信息")
	flag.Parse()
	return nil
}
