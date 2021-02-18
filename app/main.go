package main

import (
	"github.com/boggydigital/dolo"
	"log"
	"net/http"
	"net/url"
	"time"
)

func main() {
	images := []string{
		//"https://images-4.gog-statics.com/d6e63b8892c604678af5dd8b899ae46e2a8f64717d59adffdfbdfb732f3f7611.png",
		//"https://images-1.gog-statics.com/caeb7d5428193d202eeaa5ac951d291dc7f187f05d5dee7704b39a3d25488bdf.png",
		"https://images-1.gog-statics.com/11d10fea70ae724b93d21fbece31591e1d40b02af644b84bd378a5767ed43a33.png",
		//"https://images-1.gog-statics.com/817371f8c4ab44acb6781b4424b60c093cbb34b5b7346da949caa6bdc1417a2d.png",
		//"https://images-4.gog-statics.com/bd0f6859280039ecfe01bdf3fb55114be7449d90aa54ad08b7dc71bf0d27c468.png",
		//"https://images-2.gog-statics.com/a4b4a072ce88cb4c8b8a34a45ed283e1a773955c1992f03a83531a66223c15cb.png",
		//"https://images-4.gog-statics.com/799e7f0895267fb3ba028fd57380757704f00744676541acafabdb4b2c924ddf.png",
		//"https://images-4.gog-statics.com/5bac91fe4d69aace8fdcc847a877441e52ab14a6edcba20c58691a9974437ea6.png",
		//"https://images-4.gog-statics.com/aa2136185a68f0444ec2b83ce78a20f35de9f713dfb26ecb74a9af05fecdb4a2.png",
		//"https://images-2.gog-statics.com/c7d4ec19d66e5d55016d29fca440bcbe94f14f3f217bd4496c4db31d48a6a8cf.png",
		//"https://images-4.gog-statics.com/657f90a98a7789e022f86e9e54125c02b53e5bb76f71ad5c22c6ec1b83e841a8.png",
		//"https://images-3.gog-statics.com/60fb602b339485da844182f254714a429c7d12e4a0d7dd8293a9ff9db57d39ba.png",
		//"https://images-1.gog-statics.com/b2ececd37d64b98462d369cab87663208efadc8fbc135fcbc0bd8b4f221707f7.png",
		//"https://images-1.gog-statics.com/3817564662d0a180472a9d14b3c1ead9861091628b611b54c6be6f8045989dbf.png",
		//"https://images-3.gog-statics.com/1a4d265cc8f572bd3c92f05c31023573cda4f5d3489b03a19cf5a87177ac4b0f.png",
		//"https://images-4.gog-statics.com/fe90dedf8152977f2bfd50a3484502fe656dea81c8a9d6b95771c1bb8da91f88.png",
		//"https://images-3.gog-statics.com/ed02765777e869d7a10702d14d884fc2e9a1d8558eb848fdcac8ff256a2a3360.png",
		//"https://images-1.gog-statics.com/07b1af5aafdbbb1c3d216723a65bd7a465b631858bdf39e4a1ab1f71f4dc5a23.png",
		//"https://images-3.gog-statics.com/740466eb571f5637efa8292105fd1025c5188f1c03b0a40956da2a758608aa94.png",
		//"https://images-2.gog-statics.com/4db24fc08808b7d40c506fb08ea24fdf4944bddc256235c17e61374407088c60.png",
	}

	opts := &dolo.ClientOptions{
		Retries:            1,
		CheckContentLength: false,
		ResumeDownloads:    true,
		Verbose:            false,
	}
	dc := dolo.NewClient(&http.Client{
		Timeout: 10 * time.Second,
	}, nil, opts)

	for _, img := range images {
		imgUrl, err := url.Parse(img)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("downloading", imgUrl.String())

		err = dc.Download(imgUrl, "")
		if err != nil {
			log.Println(err)
		}

	}
}
