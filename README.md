# dumper
Simple tool for looking at data

Isaac Oates showed me this tool when we worked together at Etsy. This is a port of it to go.

You can use it as, is with this schema:

--DROP TABLE public.author;

CREATE TABLE public.author
(
  author_id integer NOT NULL,
  name character varying(50),
  created_at timestamp default NOW(),
  CONSTRAINT author_pkey PRIMARY KEY (author_id)
);

insert into author values (1,'Ross');
insert into author values (2,'Bill');

--DROP TABLE public.book;

CREATE TABLE public.book
(
  book_id integer NOT NULL,
  name character varying(50),
  author_id integer,
  created_at timestamp default NOW(),
  CONSTRAINT book_pkey PRIMARY KEY (book_id)
);

insert into book values (1,'Payments, a history', 1);
insert into book values (2,'The true history of payments', 2);
