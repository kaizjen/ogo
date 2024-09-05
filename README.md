# [Тестовое задание](https://gist.github.com/softzilla/92d282063f5f55393a2a8352f17792ce)
## Попробовать
**Запустите `go run ./main.go [-N <N - макс. количество задач исполняемых за раз, по дефолту 10>]`**

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

## Тесты
Поскольку `executor.go` только один и глобальный*, тесты нужно запускать с флагом `-p 1`, чтобы они не шли параллельно, т.к. иначе некоторые задания будут в очереди, когда они не должны быть в очереди (в теории можно решить это, используя какой-нибудь WaitGroup, но уже неважно)

`endpoints_test.go` - нормальный, работает окей
`executor_test.go` - абсолютный ад, сам тест очень хрупкий, а все из-за того, что `executor.go` не делает никаких гарантий о времени выполнения вещей (поэтому там есть `epsilon`, который добавляется к каждому `time.Sleep()`, потому что всякие `execute` могут завершить свою работу на пару циклов позже, и из-за такого не получается правильно оценить, что задание 3 выполнено спустя 2.4 секунды, к примеру)

Еще mutex'ы добавляют больше проблем потому что когда .Lock() в горутине выполнится - тоже неизвестно.

Сам способ тестирования такого приложения я, может, выбрал неоптимальный, но все важные функции программы (включая таймауты) ими покрыты.

Насчет `executor_test.go`, он запускает 4 задания (каждое имеет n и I больше, чем предыдущее) с N = 2, таким образом, 2 задания в очереди, остальные 2 выполнятся одно за другим. Как еще протестировать таймауты, честно, не знаю. Там хорошо бы все задания в слайс переместить, и дальше алгоритмом проверять, что нужно, но тест уже на 29 строчек больше самого файла, который он тестирует...

*вроде так делать не хорошо (не SOLID?), но это один сервер, этот модуль не импортируется, так что я думаю, что усложнять это не надо. Тем более, если нужно, это сделать очень просто, а никаких основных проблем с тестами это не решит (я уже слишком сильно изменил исходный код из-за тестов)