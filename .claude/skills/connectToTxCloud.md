# Skill: Connect to Tencent Cloud Server

## Connection Details
- **IP**: 106.54.50.88
- **Port**: 22
- **User**: ubuntu
- **Key**: `backend/deploy/test.pem`

## SSH Command
```bash
ssh -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedKeyTypes=+ssh-rsa -o StrictHostKeyChecking=no -i F:/ItsyourTurnMy/backend/deploy/test.pem ubuntu@106.54.50.88
```

## Execute Remote Command
```bash
ssh -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedKeyTypes=+ssh-rsa -o StrictHostKeyChecking=no -i F:/ItsyourTurnMy/backend/deploy/test.pem ubuntu@106.54.50.88 "<command>"
```

## Execute Command as Root
```bash
ssh -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedKeyTypes=+ssh-rsa -o StrictHostKeyChecking=no -i F:/ItsyourTurnMy/backend/deploy/test.pem ubuntu@106.54.50.88 "sudo bash -c '<command>'"
```

## SCP File Transfer
```bash
# Upload file to server
scp -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedKeyTypes=+ssh-rsa -o StrictHostKeyChecking=no -i F:/ItsyourTurnMy/backend/deploy/test.pem <local_file> ubuntu@106.54.50.88:<remote_path>

# Download file from server
scp -o HostKeyAlgorithms=+ssh-rsa -o PubkeyAcceptedKeyTypes=+ssh-rsa -o StrictHostKeyChecking=no -i F:/ItsyourTurnMy/backend/deploy/test.pem ubuntu@106.54.50.88:<remote_file> <local_path>
```

## Project Location on Server
- **Path**: `/root/ItsyourTurnMy`
- **Note**: Requires `sudo` to access /root directory

## Docker Operations
```bash
# Check containers
ssh ... ubuntu@106.54.50.88 "sudo docker ps"

# Rebuild and restart
ssh ... ubuntu@106.54.50.88 "sudo bash -c 'cd /root/ItsyourTurnMy && docker compose down && docker compose up -d --build'"

# View logs
ssh ... ubuntu@106.54.50.88 "sudo docker logs finance-backend"
ssh ... ubuntu@106.54.50.88 "sudo docker logs finance-frontend"
```

## Important Notes
1. The server has network issues accessing GitHub directly - use SCP to transfer files
2. Always use `sudo` when accessing files in `/root/`
3. SSH options are required for RSA key compatibility:
   - `-o HostKeyAlgorithms=+ssh-rsa`
   - `-o PubkeyAcceptedKeyTypes=+ssh-rsa`
