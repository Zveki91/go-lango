create table if not exists users
(
    id              serial  not null primary key,
    email           varchar not null unique,
    username        varchar not null unique,
    followers_count int     not null default 0 check ( followers_count >= 0 ),
    followees_count  int     not null default 0 check ( followees_count >= 0 )
);


create table if not exists follows
(
    follower_id int not null,
    followee_id  int not null,
    primary key (follower_id, followee_id)
);


insert into users (email, username)
values ('john@email.com', 'john'),
       ('joan@email.com', 'joan');