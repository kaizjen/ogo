# [Тестовое задание](https://gist.github.com/softzilla/92d282063f5f55393a2a8352f17792ce)
## Попробовать
**Запустите `./main.go [-N <N - макс. количество задач исполняемых за раз, по дефолту 10>]`**

Для новой задачи эндпоинт `:3000/enqueue`, принимающий все параметры от query string

`:3000/list` возвращает все задачи в JSON-формате, отсортированные по статусу (сначала выполненные, а затем по ID)

Однако для удобства тестирования есть `./index.html`, его просто открыть в браузере, он не подается с сервера. (Порт 3000 захардкоден)

Там еще есть прикольная кнопка "Randomize", которая поставит случайные значения параметров и отправит их.

Сервер устойчив к неверным параметрам и выдает ошибку (хотя без правильного HTTP кода)

## Имплементация
Он использует mutex-ы вместо каких-нибудь синхронизирующих каналов, потому что по моему мнению так намного проще, хотя и просто ошибиться (и допустим забыть разлочить мютекс), но все же.  
А именно mutex-ов два, один отвечает за очередь и список выполняемых задач (потому что их и так почти всегда меняют вместе), а второй за список выполненных задач, ждущих когда закончится TTL.

После последней итерации он еще ждет `Interval` перед тем, как "закончить" задачу - не знаю, надо такое, или нет, но от этого легко избавиться.

Для выполненных и выполняемых задач код использует map-ы, поэтому у задач есть ID, для очереди один slice. ID только растут.