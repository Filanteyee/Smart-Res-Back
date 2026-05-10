package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"smartresidency/internal/db"
	"smartresidency/internal/fcm"
	"smartresidency/internal/handler"
	"smartresidency/internal/middleware"
	"smartresidency/internal/mqtt"
	"smartresidency/internal/sensors"
)

func main() {
	_ = godotenv.Load()

	pool, err := db.Connect(os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("db:", err)
	}
	defer pool.Close()

	ctx := context.Background()

	if err := sensors.Seed(ctx, pool); err != nil {
		log.Printf("sensors seed: %v", err)
	} else {
		log.Println("sensors seeded")
	}

	var notifier handler.EventNotifier
	var mqttNotifier mqtt.Notifier
	if credPath := os.Getenv("FIREBASE_CREDENTIALS_PATH"); credPath != "" {
		s, err := fcm.New(ctx, credPath, pool)
		if err != nil {
			log.Printf("fcm init: %v (push disabled)", err)
		} else {
			notifier = s
			mqttNotifier = s
			log.Println("fcm initialized")
		}
	} else {
		log.Println("FIREBASE_CREDENTIALS_PATH not set — push disabled")
	}

	if url := os.Getenv("HIVEMQ_URL"); url != "" {
		cfg := mqtt.Config{
			URL:      url,
			Username: os.Getenv("HIVEMQ_USERNAME"),
			Password: os.Getenv("HIVEMQ_PASSWORD"),
			ClientID: os.Getenv("HIVEMQ_CLIENT_ID"),
		}
		sub, err := mqtt.New(cfg, pool, mqttNotifier)
		if err != nil {
			log.Printf("mqtt init: %v (subscriber disabled)", err)
		} else {
			defer sub.Close()
		}
	} else {
		log.Println("HIVEMQ_URL not set — mqtt disabled")
	}

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	r.Static("/uploads", "./uploads")

	api := r.Group("/api/v1")
	secret := os.Getenv("JWT_SECRET")

	authH := handler.NewAuthHandler(pool)
	api.POST("/auth/register", authH.Register)
	api.POST("/auth/login", authH.Login)

	priv := api.Group("/")
	priv.Use(middleware.Auth(secret))
	{
		priv.GET("/auth/me", authH.Me)
		priv.POST("/auth/refresh", authH.Refresh)

		profH := handler.NewProfileHandler(pool)
		priv.GET("/profiles/:id", profH.Get)
		priv.PUT("/profiles/:id", profH.Update)

		srH := handler.NewServiceRequestHandler(pool)
		priv.GET("/service-requests", srH.List)
		priv.POST("/service-requests", srH.Create)
		priv.PUT("/service-requests/:id", srH.UpdateStatus)
		priv.POST("/service-requests/:id/photos", srH.UploadPhoto)

		guestH := handler.NewGuestHandler(pool)
		priv.GET("/guest-access", guestH.List)
		priv.POST("/guest-access", guestH.Create)
		priv.PUT("/guest-access/:id/cancel", guestH.Cancel)

		barrierH := handler.NewBarrierHandler(pool)
		priv.GET("/barrier-logs", barrierH.List)
		priv.POST("/barrier-logs/open", barrierH.OpenBarrier)

		verH := handler.NewVerificationHandler(pool)
		priv.POST("/verification/requests", verH.Submit)
		priv.POST("/verification/requests/:id/documents", verH.UploadDocuments)
		priv.GET("/verification/requests", verH.List)
		priv.PUT("/verification/requests/:id/status", verH.UpdateStatus)

		sensorH := handler.NewSensorHandler(pool, notifier)
		priv.GET("/sensors", sensorH.ListByEntrance)
		priv.GET("/sensors/events", sensorH.ListEvents)
		priv.GET("/admin/sensors", sensorH.ListAll)
		priv.PATCH("/admin/sensors/events/:id/status", sensorH.UpdateEventStatus)
		priv.POST("/admin/sensors/events/:id/notify", sensorH.NotifyEvent)

		fcmH := handler.NewFCMTokenHandler(pool)
		priv.POST("/users/me/fcm-token", fcmH.Register)
		priv.POST("/users/me/fcm-token/delete", fcmH.Delete)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	log.Fatal(r.Run(":" + port))
}
