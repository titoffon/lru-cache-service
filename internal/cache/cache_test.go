package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrequentPutSameKey(t *testing.T) {
	capacity := 2
	defaultTTL := 2 * time.Second
	c := NewLRUCache(capacity, defaultTTL)

	ctx := context.Background()

	err := c.Put(ctx, "key1", "value1", 1*time.Second)
	require.NoError(t, err, "Put should not return an error")

	//обновление того же элемента, задав другой ttl и другое значение
	err = c.Put(ctx, "key1", "value2", 3*time.Second)
	require.NoError(t, err, "Put should not return an error")

	val, expAt, err := c.Get(ctx, "key1")
	require.NoError(t, err, "Get should not return an error for existing key")


	assert.Equal(t, "value2", val, "value should have been updated")

	//если ttl обновился то время стало больше 1 секунды
	assert.GreaterOrEqual(t, time.Until(expAt).Milliseconds(), int64(1500),
		"TTL (expiresAt) should be at least ~2s from now")
}

func TestGet (t *testing.T) {
	capacity := 2
	defaultTTL := 1 * time.Second
	c := NewLRUCache(capacity, defaultTTL)

	ctx := context.Background()

	err := c.Put(ctx, "key_exp", "some_value", 500*time.Millisecond)
	require.NoError(t, err, "Put should not return an error")

	//ждёмс
	time.Sleep(600 * time.Millisecond)

	_, _, err = c.Get(ctx, "key_exp")
	assert.Error(t, err, "expected an error (key expired)")
	assert.Equal(t, ErrKeyNotFound, err, "error should be ErrKeyNotFound for expired key")
}

func TestPutOverflow(t *testing.T) {
	capacity := 2
	defaultTTL := 5 * time.Second
	c := NewLRUCache(capacity, defaultTTL)

	ctx := context.Background()

	err := c.Put(ctx, "key1", "val1", 0)
	require.NoError(t, err)

	err = c.Put(ctx, "key2", "val2", 0)
	require.NoError(t, err)

	//добавляем 3-й ключ => должен вытеснить "key1" как LRU
	err = c.Put(ctx, "key3", "val3", 0)
	require.NoError(t, err)

	//проверяем, что key1 был удалён
	_, _, err = c.Get(ctx, "key1")
	assert.Error(t, err, "expected key1 to be evicted")
	assert.Equal(t, ErrKeyNotFound, err, "error should be ErrKeyNotFound")

	//key2 и key3 должны быть доступны
	v2, _, err2 := c.Get(ctx, "key2")
	assert.NoError(t, err2, "key2 should still exist")
	assert.Equal(t, "val2", v2, "invalid value for key2")

	v3, _, err3 := c.Get(ctx, "key3")
	assert.NoError(t, err3, "key3 should exist")
	assert.Equal(t, "val3", v3, "invalid value for key3")
}

func TestEvict(t *testing.T) {
	capacity := 2
	defaultTTL := 5 * time.Second
	c := NewLRUCache(capacity, defaultTTL)

	ctx := context.Background()

	err := c.Put(ctx, "key1", "val1", 0)
	require.NoError(t, err)

	err = c.Put(ctx, "key2", "val2", 0)
	require.NoError(t, err)

	//удаляем key2
	evictedVal, err := c.Evict(ctx, "key2")
	require.NoError(t, err, "Evict should not return an error for existing key")
	assert.Equal(t, "val2", evictedVal, "evicted value should match the one put before")

	//проверяем, что key2 удалён
	_, _, err = c.Get(ctx, "key2")
	assert.Error(t, err, "expected error for key2, since it's evicted")
	assert.Equal(t, ErrKeyNotFound, err, "error should be ErrKeyNotFound for evicted key2")

	//пробуем удалить несуществующий ключ
	_, err = c.Evict(ctx, "no_such_key")
	assert.Error(t, err, "expected error when evicting non-existing key")
	assert.Equal(t, ErrKeyNotFound, err, "error should be ErrKeyNotFound for non-existing key")
}

func TestEvictAll(t *testing.T) {
	capacity := 3
	defaultTTL := 5 * time.Second
	c := NewLRUCache(capacity, defaultTTL)

	ctx := context.Background()

	err := c.Put(ctx, "k1", 1, 0)
	require.NoError(t, err)
	err = c.Put(ctx, "k2", 2, 0)
	require.NoError(t, err)
	err = c.Put(ctx, "k3", 3, 0)
	require.NoError(t, err)

	err = c.EvictAll(ctx)
	require.NoError(t, err, "EvictAll should not return an error")

	//все ключи должны отсутствовать
	_, _, err = c.Get(ctx, "k1")
	assert.Equal(t, ErrKeyNotFound, err, "k1 must be evicted")

	_, _, err = c.Get(ctx, "k2")
	assert.Equal(t, ErrKeyNotFound, err, "k2 must be evicted")

	_, _, err = c.Get(ctx, "k3")
	assert.Equal(t, ErrKeyNotFound, err, "k3 must be evicted")
}