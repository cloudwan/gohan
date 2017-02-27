## Gohan script extension
{% raw %}

Note: This function is experimental. Any APIs are subject to change.

Gohan script is an Ansible-like MACRO language
for extending your Go code with MACRO functionality.

## Example

```yaml
extensions:
- id: order
  path: /v1.0/store/orders
  code: |
    tasks:
    - when: event_type == "post_create_in_transaction"
      blocks:
      #  Try debugger
      # - debugger:
      - db_get:
          tx: $transaction
          schema_id: pet
          id: $resource.pet_id
        register: pet
      - when: pet.status != "available"
        blocks:
        - vars:
            exception:
              name: CustomException
              code: 400
              message: "Selected pet isn't available"
        else:
        - db_update:
            tx: $transaction
            schema_id: pet
            data:
                id: $resource.pet_id
                status: "pending"
    - when: event_type == "post_update_in_transaction"
      blocks:
      - when: resource.status == "approved"
        blocks:
        - db_update:
            tx: $transaction
            schema_id: pet
            data:
              id: $resource.pet_id
              status: "sold"
```

## Standalone Example

```yaml
    vars:
      world: "USA"
      foods:
      - apple
      - orange
      - banana
    tasks:
    - debug: var=$foods[2]
    - debug: msg="Hello \" {{ world }}"
    - blocks:
        - debug: msg="I like {{ item }}"
          with_items:
        - apple
        - orange
        - banana
        - debug: msg="I like {{ item }}"
          with_items: $foods
        # Unlike Ansible, We treat a value as identifier if a string value starts with "$"
        # otherwise it is a value
        - debug: msg="{{ item.key }} likes {{ item.value }}"
          with_dict:
            Alice: apple
            Bob: orange
        - debug: msg="This shouldn't be called"
          when: 1 == 0
          else:
          - debug: msg="This should be called"
        - fail: "failed"
          when: 1 == 1
          rescue:
          - debug: msg="rescued {{ error }}"
          always:
          - debug: msg="Drink beer!"
    - debug: msg="test {{ 1 == 1 }}"
    - include: lib.yaml
        vars:
        local_vars: hello from imported code
```

see more detail on extension/gohanscript/test/core_test.yaml

## CLI

You can run Gohan script code using this

```
gohan run ../examples/sample1.yaml
```

## Tasks

You can run list of tasks.

```yaml
  tasks:
  - debug: msg="Hello World"
  - debug: msg="This is gohan script"
```

save this file to hello_world.yaml.

```
$ gohan run hello_world.yaml
15:17:07.029 ▶ DEBUG  hello_world.yaml:1: Hello World
15:17:07.029 ▶ DEBUG  hello_world.yaml:2: This is gohan script
```

## Variables

You can define variables using "vars".

```yaml
  tasks:
  - vars:
      place: "Earth"
      person:
        name: "John"
        age: "30"
  - debug: msg="Hello {{place}}"
  - debug: var=$place
  - debug: msg="Hello {{person.name}} "
  - debug: var=$person.name
  - debug: # show everything
```

Any string including "{{" get considered as django template. so
you can use variables in their. if string start with $, it get considered as
a variable identifier.
(We are using pongo2 which supports subset of django template..)

```
    $ gohan run variable.yaml
    15:21:43.090 ▶ DEBUG  variable.yaml:6 Hello Earth
    15:21:43.091 ▶ DEBUG  variable.yaml:7 Earth
    15:21:43.091 ▶ DEBUG  variable.yaml:8 Hello John
    15:21:43.091 ▶ DEBUG  variable.yaml:9 John
    15:21:43.091 ▶ DEBUG  variable.yaml:10 Dump vars
    15:21:43.091 ▶ DEBUG      person: map[name:John age:30]
    15:21:43.091 ▶ DEBUG      __file__: variable.yaml
    15:21:43.091 ▶ DEBUG      __dir__: .
    15:21:43.091 ▶ DEBUG      place: Earth
```

## Loops

You can loop over the list item.

```yaml
    vars:
        foods:
        - apple
        - orange
        - banana
    tasks:
    - debug: msg="{{ item }}"
      with_items:
      - apple
      - orange
      - banana
    - debug: msg="{{ item }}"
      with_items: $foods
```

```
    $ gohan run with_items.yaml
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 apple
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 orange
    15:28:47.736 ▶ DEBUG  with_items.yaml:6 banana
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 apple
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 orange
    15:28:47.736 ▶ DEBUG  with_items.yaml:11 banana
```

You can also loop over a dict.

```yaml
    vars:
    person:
        name: "John"
        age: "30"
    tasks:
    - debug: msg="{{ item.key }} {{ item.value }}"
      with_dict:
        name: "John"
        age: "30"
    - debug: msg="{{ item.key }} {{ item.value }}"
      with_dict: $person
```

```
    $ gohan run with_items.yaml
    15:32:42.513 ▶ DEBUG  with_items.yaml:5 name John
    15:32:42.513 ▶ DEBUG  with_items.yaml:5 age 30
    15:32:42.513 ▶ DEBUG  with_items.yaml:9 name John
    15:32:42.513 ▶ DEBUG  with_items.yaml:9 age 30
```

```yaml
    tasks:
    - vars:
        result: ""
        persons:
        - name: Alice
          hobbies:
          - mailing
          - reading
        - name: Bob
          hobbies:
          - mailing
          - running
    - blocks:
        - vars:
            result: "{{result}}{{item}}"
          with_items: $person.hobbies
      with_items: $persons
      loop_var: person
```

## Conditional

You can use "when" for conditional.
You can use "else" blocks with "when".

```yaml
    vars:
      number: 1
    tasks:
    - debug: msg="Should be called"
      when: number == 1
    - debug: msg="Should not be called"
      when: number == 0
      else:
      - debug: msg="Should be called"
```

```
    $ gohan run when.yaml
    15:35:55.358 ▶ DEBUG  when.yaml:3 Should be called
```

## Retry

You can retry task.

- retry: how many times you will retry a task
- delay: how many seconds you will wait on next retry

```yaml
    tasks:
    - fail: msg="Failed"
      retry: 3
      delay: 3
```

```
    $ gohan run retry.yaml
    15:43:35.720 ▶ WARNING  error: tasks[0]: Failed
    15:43:35.720 ▶ WARNING  error: tasks[0]: Failed
    Failed
```

## Blocks

You can group a set of tasks using blocks.
blocks also supports loops, conditional and retries.

```yaml
    tasks:
    - blocks:
      - debug: msg="hello"
      - debug: msg="from in block"
```

```
    $ gohan run blocks.yaml
    15:48:30.231 ▶ DEBUG  blocks.yaml:2 hello
    15:48:30.231 ▶ DEBUG  blocks.yaml:3 from in block
```

## Register

You can change variable value using "register".

```yaml
    tasks:
    - http_get: url=https://status.github.com/api/status.json
      register: result
    - debug: msg="{{result.contents.status}}"
```

```
    $ gohan run register.yaml
    15:51:11.005 ▶ DEBUG  [register.yaml line:3 column:2] good
```

## Concurrency

We support concurrent execution over a loop.

- worker: specify number of max workers

```yaml
    tasks:
    - blocks:
      - http_get: url="https://status.github.com/{{ item }}"
        register: result
      - debug: var=$result.raw_body
    worker: 3
    with_items:
    - /api/status.json
    - /api.json
    - /api/last-message.json
```

```
    $ gohan run worker.yaml
    15:58:49.151 ▶ DEBUG  worker.yaml:4 {"status_url":"https://status.github.com/api/status.json","messages_url":"https://status.github.com/api/messages.json","last_message_url":"https://status.github.com/api/last-message.json","daily_summary":"https://status.github.com/api/daily-summary.json"}
    15:58:49.156 ▶ DEBUG  worker.yaml:4 {"status":"good","body":"Everything operating normally.","created_on":"2016-03-03T22:03:59Z"}
    15:58:49.156 ▶ DEBUG  worker.yaml:4 {"status":"good","last_updated":"2016-03-08T23:58:27Z"}
```

You can also execute tasks in background.

```yaml
    tasks:
    - background:
      - sleep: 1000
      - debug: msg="called 2"
    - debug: msg="called 1"
    - sleep: 2000
    - debug: msg="called 3"
```

```
    $ gohan run background.yaml
    16:02:55.034 ▶ DEBUG  background.yaml:6 called 1
    16:02:56.038 ▶ DEBUG  background.yaml:4 called 2
    16:02:57.038 ▶ DEBUG  background.yaml:8 called 3
```


### Define function

You can define function using "define" task.

- name: name of function
- args: arguments
- body: body of code

```yaml
    tasks:
    - define:
        name: fib
        args:
          x: int
        body:
        - when: x < 2
          return: x
        - sub_int: a=$x b=1
          register: $x
        - fib:
            x: $x
          register: a
        - sub_int: a=$x b=1
          register: x
        - fib:
            x: $x
          register: b
        - add_int: a=$a b=$b
          register: result
        - return: result
    - fib: x=10
      register: result2
    - debug: msg="result = {{result2}}"
```

you can use return task in function block.

```
    $ gohan run fib.yaml
    16:07:39.964 ▶ DEBUG  fib.yaml:23 result = 55
```

### Include

You can include gohan script

```yaml
    tasks:
    - include: lib.yaml
```

```
    $ gohan run include.yaml
    16:11:20.569 ▶ DEBUG  lib.yaml:0 imported
```

## Debugger mode

You can set breakpoint using "debugger"

```yaml
    vars:
    world: "USA"
    foods:
    - apple
    - orange
    - banana
    tasks:
    - debugger:
    - debug: msg="Hello {{ world }}"
```

```
    10:50:55.052 gohanscript INFO  Debugger port: telnet localhost 40000
```


Supported command in debugger mode:

- s: step task
- n: next task
- r: return
- c: continue
- p: print current context
- p code: execute miniGo
- l: show current task

You will get separate port per go routine.

## Command line argument

additional arguments will be stored in variable.
If the value doesn't contain "=", it will be pushed to args.
If the value contains "=", it get splitted for key and value and stored in flags.

```yaml
    tasks:
    - debug: msg="{{ flags.greeting }} {{ args.0 }}"
```

```yaml
$ gohan run args.yaml world greeting=hello
hello world
```

## Run test

You can run gohan build-in test. gohan test code find test gohan code
in the specified directory.

```
    gohan test
```


## Run Gohan script from Go

```go
	vm := gohan.NewVM()
	_, err := vm.RunFile("test/spec.yaml")
	if err != nil {
		t.Error(err)
	}
```

## Add new task using Go

You can auto generate adapter functions using ./extension/gohanscript/tools/gen.go.

```
  go run ./extension/gohanscript/tools/gen.go genlib -t extension/gohanscript/templates/lib.tmpl -p github.com/cloudwan/gohan/extension/gohanscript/lib -e autogen -ep extension/gohanscript/autogen
```


## More examples and supported functions

please take a look

- extension/gohanscript/lib/tests
- extension/gohanscript/tests

{% endraw %}