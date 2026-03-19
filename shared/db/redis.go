package db

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"tt.tracker/shared/models"
)

func NewRedisClient(addr, password string) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, fmt.Errorf("ping redis: %w", err)
	}
	return client, nil
}

func GetLastHistoryIdx(ctx context.Context, client *redis.Client, server string, vrpID int) (int, error) {
	key := fmt.Sprintf("%s:player:%d:hidx", server, vrpID)
	val, err := client.Get(ctx, key).Int()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

func SetLastHistoryIdx(ctx context.Context, client *redis.Client, server string, vrpID, idx int) error {
	key := fmt.Sprintf("%s:player:%d:hidx", server, vrpID)
	return client.Set(ctx, key, idx, 5*time.Minute).Err()
}

func WritePlayerState(ctx context.Context, client *redis.Client, server string, player *models.Player, extraTrail []models.Position) error {
	prefix := fmt.Sprintf("%s:player:%d", server, player.VrpID)

	pipe := client.Pipeline()

	// HSET current state
	pipe.HSet(ctx, prefix+":current", map[string]interface{}{
		"name":         player.Name,
		"vrp_id":       player.VrpID,
		"x":            player.Position.X,
		"y":            player.Position.Y,
		"z":            player.Position.Z,
		"job_group":    player.Job.Group,
		"job_name":     player.Job.Name,
		"vehicle_type": player.Vehicle.Type,
		"vehicle_name": player.Vehicle.Name,
	})
	pipe.Expire(ctx, prefix+":current", 5*time.Minute)

	// Build trail entries: extra history (oldest→newest), then current position
	trailKey := prefix + ":trail"
	var entries []interface{}
	for _, t := range extraTrail {
		entries = append(entries, fmt.Sprintf("%f,%f,%f", t.X, t.Y, t.Z))
	}
	entries = append(entries, fmt.Sprintf("%f,%f,%f", player.Position.X, player.Position.Y, player.Position.Z))
	pipe.RPush(ctx, trailKey, entries...)

	pipe.LTrim(ctx, trailKey, -60, -1)
	pipe.Expire(ctx, trailKey, 5*time.Minute)

	_, err := pipe.Exec(ctx)
	return err
}

func GetAllPlayers(ctx context.Context, client *redis.Client, server string) ([]models.PlayerState, error) {
	pattern := fmt.Sprintf("%s:player:*:current", server)
	var players []models.PlayerState

	iter := client.Scan(ctx, 0, pattern, 200).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		data, err := client.HGetAll(ctx, key).Result()
		if err != nil || len(data) == 0 {
			continue
		}

		ps := models.PlayerState{
			Name:        data["name"],
			JobGroup:    data["job_group"],
			JobName:     data["job_name"],
			VehicleType: data["vehicle_type"],
			VehicleName: data["vehicle_name"],
		}
		ps.VrpID, _ = strconv.Atoi(data["vrp_id"])
		ps.X, _ = strconv.ParseFloat(data["x"], 64)
		ps.Y, _ = strconv.ParseFloat(data["y"], 64)
		ps.Z, _ = strconv.ParseFloat(data["z"], 64)

		// Get trail
		trailKey := strings.Replace(key, ":current", ":trail", 1)
		trailData, err := client.LRange(ctx, trailKey, 0, 59).Result()
		if err == nil {
			for _, entry := range trailData {
				parts := strings.Split(entry, ",")
				if len(parts) == 3 {
					x, _ := strconv.ParseFloat(parts[0], 64)
					y, _ := strconv.ParseFloat(parts[1], 64)
					z, _ := strconv.ParseFloat(parts[2], 64)
					ps.Trail = append(ps.Trail, models.Position{X: x, Y: y, Z: z})
				}
			}
		}

		players = append(players, ps)
	}
	if err := iter.Err(); err != nil {
		return nil, err
	}

	return players, nil
}

func GetPlayerTrail(ctx context.Context, client *redis.Client, server string, vrpID string) ([]models.Position, error) {
	key := fmt.Sprintf("%s:player:%s:trail", server, vrpID)
	data, err := client.LRange(ctx, key, 0, 59).Result()
	if err != nil {
		return nil, err
	}

	var trail []models.Position
	for _, entry := range data {
		parts := strings.Split(entry, ",")
		if len(parts) == 3 {
			x, _ := strconv.ParseFloat(parts[0], 64)
			y, _ := strconv.ParseFloat(parts[1], 64)
			z, _ := strconv.ParseFloat(parts[2], 64)
			trail = append(trail, models.Position{X: x, Y: y, Z: z})
		}
	}
	return trail, nil
}
