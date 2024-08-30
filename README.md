# Project Description
This project is a Go-based database system designed to manage users and their subscriptions. It includes functionalities for creating, retrieving, updating, and deleting users, as well as managing their subscription statuses and traffic usage.

## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [Scheduler](#scheduler)
- [Docker](#docker)
- [Contributing](#contributing)

## Features
- User management (create, retrieve, update, delete)
- Subscription management (update status, check status)
- Traffic management (update traffic, reset traffic)
- Scheduled tasks for resetting traffic and checking subscriptions
- Authentication middleware for API endpoints
- CORS configuration for API access

## Installation
### Clone the repository:
git clone https://github.com/YuarenArt/tg-users-database.git



### Install dependencies:
go mod download



### Set up environment variables:
Create a `.env` file in the root directory with the following content:
BOT_TOKEN=your_bot_token
DB_USER=your_db_user
DB_PASSWORD=your_db_password
DB_NAME=your_db_name
DB_SSLMODE=disable
HOST=your_db_host
PORT=your_db_port



### Build the project:
go build -o main .



## Usage
Run the application:
./main


The application will start on port 8082 by default.

## API Endpoints
The following API endpoints are available:
- `POST /users`: Create a new user
- `GET /users/:username`: Retrieve a user by username
- `PUT /users/:username`: Update a user's subscription
- `DELETE /users/:username`: Delete a user by username
- `GET /users/:username/subscription`: Get a user's subscription status
- `GET /users/:username/exists`: Check if a user exists
- `PUT /users/:username/traffic`: Update a user's traffic

## Scheduler
The project includes a scheduler that performs the following tasks:
- Reset traffic for all users weekly
- Check and update subscriptions daily

The scheduler is implemented using the `robfig/cron` package.

## Docker
The project includes a Dockerfile for building and running the application in a container.

### Build the Docker image:
docker build -t tg-users-database .



### Run the Docker container:
docker run -p 8082:8082 --env-file .env tg-users-database



## Contributing
Contributions are welcome! Please open an issue or submit a pull request.
