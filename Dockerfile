FROM golang:1.6
RUN curl -fsSL "https://drive.google.com/uc?export=download&id=0B4y35FiV1wh7cENtOXlicTFaRUE" -o mecab.tar.gz \
    && tar zxfv mecab.tar.gz \
    && cd mecab-0.996 \
    && ./configure --enable-utf8-only \
    && make && make check && make install && ldconfig \
    && cd .. \
    && rm -rf mecab-0.996 && rm mecab.tar.gz \
    && curl -fsSL "https://drive.google.com/uc?export=download&id=0B4y35FiV1wh7MWVlSDBCSXZMTXM" -o mecab-ipadic.tar.gz \
    && tar zxfv mecab-ipadic.tar.gz \
    && cd mecab-ipadic-2.7.0-20070801 \
    && ./configure --with-charset=utf8 \
    && make && make install \
    && cd .. \
    && rm -rf mecab-ipadic-2.7.0-20070801 && rm mecab-ipadic.tar.gz
ENV CGO_LDFLAGS "-L/usr/local/lib -lmecab -lstdc++"
ENV CGO_FLAGS "-I/usr/local/include"

RUN mkdir -p /go/src/app

RUN apt-get update && apt-get install -y xz-utils patch file \
    && git clone --depth 1 https://github.com/neologd/mecab-ipadic-neologd.git && cd mecab-ipadic-neologd && ./bin/install-mecab-ipadic-neologd --forceyes --asuser \
    && echo "dicdir: $(mecab-config --dicdir)/mecab-ipadic-neologd\nversion: $(git log -n 1 --pretty=tformat:%H)" > /go/src/app/neologd-config.yml \
    && cd .. && rm -rf mecab-ipadic-neologd

WORKDIR /go/src/app

COPY . /go/src/app
RUN go-wrapper download
RUN go-wrapper install

CMD ["go-wrapper", "run"]
