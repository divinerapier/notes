# React

## Components

### Prop.Childern

有两个嵌套的 `components`

``` typescript
<CreateForm onCancel={() => handleModalVisible(false)} modalVisible={createModalVisible}>
  <ProPost<PostListItem, PostListItem>
    onSubmit={async (value) => {
      const success = await handleAdd(value);
      if (success) {
        handleModalVisible(false);
        if (actionRef.current) {
          actionRef.current.reload();
        }
      }
    }}
    rowKey="key"
    type="form"
    columns={columns}
    rowSelection={{}}
  />
</CreateForm>
```

`Component: CreateForm` 

``` typescript
const CreateForm: React.FC<CreatePostProps> = (props) => {
  const { modalVisible, onCancel } = props;

  return (
    <Modal
      destroyOnClose
      title="新建规则"
      visible={modalVisible}
      onCancel={() => onCancel()}
      footer={null}
    >
      {props.children}
    </Modal>
  );
};
```

中 `{props.childern}` 就是 `<CreateForm></CreateForm>` 中的内容。

## Umi

### connect

在使用 `connect` 关联 `model` 与 `component` 时，要求 `model` 文件的目录名，`model` 的 `namespace` 及 `connect` 的参数名称，三者要一致(我不懂为啥，我也不知道对不对，反正这样做就通过了，改一个字母都报错)。

[参考](https://www.cnblogs.com/wisewrong/p/12186662.html)

## Problems

### ENOSPC: System limit for number of file watchers reached

``` bash
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p
```

https://stackoverflow.com/questions/55763428/react-native-error-enospc-system-limit-for-number-of-file-watchers-reached
