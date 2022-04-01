CREATE table if not exists posts
(
    id         serial      not null primary key,
    user_id    int         not null references users (id),
    content    varchar     not null,
    spoiler_of varchar,
    nsfw       bool        not null,
    created_at timestamptz not null default now()
);

create table if not exists timeline
(
    id      serial not null primary key,
    user_id int    not null references users (id),
    post_id int    not null references posts (id)
);


--Indexes
create index if not exists sorted_posts on posts (created_at DESC );
create unique index if not exists timeline_unique ON timeline (user_id,post_id);
