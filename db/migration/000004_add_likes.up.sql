alter table posts add likes_count int not null
    default 0 check ( likes_count >= 0 );

create table if not exists post_likes (
    user_id int not null references users(id),
    post_id int not null references posts(id),
    PRIMARY KEY (user_id,post_id)
);