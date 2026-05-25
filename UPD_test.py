import socket
# 创建一个 UDP 套接字
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
# 向本机的 3000 端口发送一条测试消息
message = b"He llo, UDP Server!" # 注意: 字符串需要编码为字节
server_address = ("127.0.0.1", 3000)
sock.sendto(message, server_address)
sock.close()
print("UDP 数据包已发送！")