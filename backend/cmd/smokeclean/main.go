package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// 清理冒烟测试数据（冒烟测试家庭 + 其宝宝和记录 + 冒烟测试用户）
func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := pgx.Connect(ctx, "postgres://postgres:123456@192.168.1.8:5432/baby_care?sslmode=disable")
	if err != nil {
		fmt.Println("连接失败:", err)
		return
	}
	defer conn.Close(ctx)

	// 找到冒烟测试用户创建的家庭
	var familyID int64
	err = conn.QueryRow(ctx, `SELECT id FROM families WHERE name = '冒烟测试家庭' ORDER BY id DESC LIMIT 1`).Scan(&familyID)
	if err != nil {
		fmt.Println("无冒烟测试家庭，无需清理")
		return
	}

	tag, err := conn.Exec(ctx, `DELETE FROM care_records WHERE family_id = $1`, familyID)
	fmt.Println("删除记录:", tag.RowsAffected(), err)
	tag, err = conn.Exec(ctx, `DELETE FROM babies WHERE family_id = $1`, familyID)
	fmt.Println("删除宝宝:", tag.RowsAffected(), err)
	tag, err = conn.Exec(ctx, `DELETE FROM family_members WHERE family_id = $1`, familyID)
	fmt.Println("删除成员:", tag.RowsAffected(), err)
	tag, err = conn.Exec(ctx, `DELETE FROM families WHERE id = $1`, familyID)
	fmt.Println("删除家庭:", tag.RowsAffected(), err)
	tag, err = conn.Exec(ctx, `DELETE FROM users WHERE nickname LIKE '冒烟%'`)
	fmt.Println("删除用户:", tag.RowsAffected(), err)
	fmt.Println("清理完成")
}
