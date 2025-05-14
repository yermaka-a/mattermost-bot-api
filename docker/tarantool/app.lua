-- Задаем порт прослушивания
box.cfg{
    listen = 3301
}

 box.schema.space.create('users', {if_not_exists = true})

box.space.users:format({
		{name = 'id', type = 'string'},
		{name = 'votes', type = 'map'}
	})


box.space.users:create_index('primary', {
			type = 'hash',
			parts = {{field = 'id', type = 'string'}},
			unique = true,
			if_not_exists = true
		})

box.schema.space.create('voting', {if_not_exists = true})

box.space.voting:format({
            {name = 'id', type = 'string'},
            {name = 'creator_id', type = 'string'},
            {name = 'question', type = 'string'},
            {name = 'options', type = 'array'},
            {name = 'votes', type = 'array'},
            {name = 'created_at', type = 'integer'},
            {name = 'is_active', type = 'boolean'},
            {name = 'expires_at', type = 'integer'}
        })

box.space.voting:create_index('primary', {
           	type = 'hash',
			parts = {{field = 'id', type = 'string'}},
			unique = true,
            if_not_exists = true
        })