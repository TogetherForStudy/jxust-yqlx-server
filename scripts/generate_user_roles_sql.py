#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
生成 user_roles 表的 SQL 导入脚本
user_id 从 1 到 11432，role_id 全部为 1
"""

def generate_sql_insert_script(start_user_id=1, end_user_id=11432, role_id=1, output_file='user_roles_insert.sql'):
    """
    生成 SQL INSERT 语句
    
    Args:
        start_user_id: 起始用户ID
        end_user_id: 结束用户ID
        role_id: 角色ID（默认为1）
        output_file: 输出文件名
    """
    sql_statements = []
    
    # 生成批量插入语句（每1000条一个INSERT语句，提高导入效率）
    batch_size = 1000
    total_records = end_user_id - start_user_id + 1
    
    print(f"正在生成 SQL 脚本...")
    print(f"用户ID范围: {start_user_id} - {end_user_id}")
    print(f"角色ID: {role_id}")
    print(f"总记录数: {total_records}")
    
    for batch_start in range(start_user_id, end_user_id + 1, batch_size):
        batch_end = min(batch_start + batch_size - 1, end_user_id)
        values = []
        
        for user_id in range(batch_start, batch_end + 1):
            values.append(f"({user_id}, {role_id}, NOW(), NOW())")
        
        sql = f"INSERT INTO `user_roles` (`user_id`, `role_id`, `created_at`, `updated_at`) VALUES\n"
        sql += ",\n".join(values) + ";\n"
        sql_statements.append(sql)
    
    # 写入文件
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write("-- 用户角色关联表数据导入脚本\n")
        f.write(f"-- 生成时间: {__import__('datetime').datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
        f.write(f"-- 用户ID范围: {start_user_id} - {end_user_id}\n")
        f.write(f"-- 角色ID: {role_id}\n")
        f.write(f"-- 总记录数: {total_records}\n\n")
        f.write("-- 开始事务（可选，建议在导入前开启）\n")
        f.write("-- START TRANSACTION;\n\n")
        
        for sql in sql_statements:
            f.write(sql)
            f.write("\n")
        
        f.write("-- 提交事务（如果开启了事务）\n")
        f.write("-- COMMIT;\n")
    
    print(f"\nSQL 脚本已生成: {output_file}")
    print(f"共生成 {len(sql_statements)} 个 INSERT 语句块")
    print(f"每个语句块最多包含 {batch_size} 条记录")


if __name__ == "__main__":
    # 生成 SQL 脚本
    generate_sql_insert_script(
        start_user_id=1,
        end_user_id=11432,
        role_id=1,
        output_file='user_roles_insert.sql'
    )

