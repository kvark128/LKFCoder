# LKFCoder

LKFCoder — это утилита командной строки, предназначенная для кодирования и декодирования аудиокниг специализированного формата lkf.

## Сборка
Для получения исполняемого файла из исходного кода выполните следующую команду:

	go install github.com/kvark128/LKFCoder

## Использование
Первый аргумент утилиты задаёт требуемое действие: decode или encode.

* decode - указывает, что LKFCoder должен декодировать lkf-файлы в формат mp3.
* encode - указывает, что LKFCoder должен кодировать mp3-файлы в формат lkf.

Второй аргумент задаёт путь к файлу или каталогу, который требуется обработать.
Если он не указан, то используется текущий рабочий каталог.
При указании пути к каталогу, будут обработаны все файлы во всех существующих подкаталогах.
Обрабатываемые файлы определяются по расширению. lkf декодируется в mp3 или mp3 кодируется в lkf.

Например, если книга в формате lkf находится по пути C:\MyBook, то для её преобразования в формат mp3 следует выполнить следующую команду:

	LKFCoder decode C:\MyBook

Результат работы записывается в исходный файл, у которого по окончанию меняется расширение.
