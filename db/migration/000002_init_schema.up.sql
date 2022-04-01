alter table follows add foreign key (followee_id) references users;
alter table follows add foreign key (follower_id) references users;
alter table users add avatar_url varchar