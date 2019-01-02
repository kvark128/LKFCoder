# LKFCoder

LKFCoder — это утилита написанная на языке go, предназначенная для кодирования и декодирования аудиокниг формата lkf.

## Сборка
Утилита не имеет сторонних зависимостей и собирается на любой платформе где доступен инструментарий go. Для получения исполняемого файла из исходного кода выполните следующую команду:

	go get github.com/kvark128/LKFCoder

## Использование
Первый аргумент программы указывает требуемое действие: decode, encode или version.

* decode - указывает, что LKFCoder должен декодировать lkf-файлы в формат mp3.
* encode - указывает, что LKFCoder должен кодировать mp3-файлы в формат lkf.
* version - показывает версию LKFCoder.

Второй аргумент указывает путь к файлу или каталогу, который требуется обработать.
Если второй аргумент не указан, то используется текущий рабочий каталог.
При указании каталога будут обработаны все файлы во всех его подкаталогах.
Обрабатываемые файлы определяются по расширению. lkf декодируется в mp3 или mp3 кодируется в lkf.

Например, если книга в формате lkf находится по пути C:\MyBook, то для её преобразования в формат mp3 выполните следующую команду:

	LKFCoder decode C:\MyBook

Результат работы записывается в исходный файл, у которого по окончанию меняется расширение.
