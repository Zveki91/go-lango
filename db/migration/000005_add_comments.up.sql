Create table if not exists comments
(
    id          serial      not null primary key,
    user_id     int         not null references users (id),
    post_id     int         not null references posts (id),
    content     varchar     not null,
    likes_count int         not null default 0 check ( likes_count >= 0 ),
    created_at  timestamptz not null default now()
);

create index if not exists sorted_comments on comments (created_at desc);


create table if not exists comment_likes
(
    id         serial      not null primary key,
    comment_id int         not null references comments (id),
    post_id    int         not null references posts (id),
    user_id    int         not null references users (id),
    created_at timestamptz not null default now()
);

insert into posts(user_id, content, nsfw)
values (1, 'samplePost with shitloads of text', true);
insert into timeline(id, user_id, post_id)
values ((select last_value from posts_id_seq) +1 , 1, 1);
insert into comments (user_id, post_id, content)
VALUES (1, (select last_value from posts_id_seq), 'ovo je komentar');
