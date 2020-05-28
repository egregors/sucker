package main

import (
	"context"
	"github.com/egregors/sucker/app/downloader"
	"github.com/egregors/sucker/app/internal"
	"log"
	"os"
)

// todo:
//  - [x] мульти бары для загрузок
//  - [x] удалять бары, когда загрузилось
// 	- [ ] ловить interrupt и удалять недокаченое
//  - [ ] выводить в поплог когда максимальное количество ретраев привышено
//  - [ ] попрообвать сделать дозагрузку файла на баре после ретрая
//  - [ ] все таки как то удалять недокаченные файлы
//  - [x] придумать как сделать общий прогресс бар для всего количества закачек
//
//  все еще не понятно, что делать при ктрл - с

func main() {
	// 1. get CLI args
	// 2. get URLs list
	// 3. create wg, progress, context
	// 4. create downloader
	// 5. start downloading
	// 6. catch interrupt

	// get all pages licks from CLI args
	pageLinks, err := internal.ParseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("bad args: %v", err)
	}

	ctx, _ := context.WithCancel(context.Background())

	d, err := downloader.NewDownloader(ctx, pageLinks)
	if err != nil {
		panic(err)
	}
	d.Run()
}
