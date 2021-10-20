# cribe

Cribe is a telegram bot for downloading youtube videos.
You just send a link to the bot and it uploads the video to Telegram

## Useful
For local only. Youtube after limit download speed =( But you can try my open bot [Cribe](https://t.me/cribe_bot)


## Install
You need to read all the environment variables from the docker-compose.yml file
```bash
sudo docker-compose up
```

## Requirements
Why do I need a telegram bot proxy? To upload files larger than 20mb to telegram. You will also need to register your hash, id in telegram. It is not difficult.

* PostgreeSQL for metadata
* Redis for store user session
* RabbitMQ for fetching/uploading queue
* telegrambot api proxy for uploading large files to telegram


## Congrats
* [youtube lib](https://github.com/kkdai/youtube)
* [telegrambotapi](https://github.com/go-telegram-bot-api/telegram-bot-api)

## Dependencies

I use my own forks of the above libraries. I have solved some problems with video uploading and context cancellation in these libraries
* [youtubelib](https://github.com/robotomize/youtube)
* [telegrambotapi](https://github.com/robotomize/telegram-bot-api)

## Usage
## <img src="https://github.com/robotomize/cribe/raw/main/docs/process.gif">


## License
Cribe is under the Apache 2.0 license. See the [LICENSE](LICENSE) file for details.
## Contact
Telegram: [@robotomize](https://t.me/robotomize)
