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

## Problems

### ENOSPC: System limit for number of file watchers reached

``` bash
echo fs.inotify.max_user_watches=524288 | sudo tee -a /etc/sysctl.conf && sudo sysctl -p
```

https://stackoverflow.com/questions/55763428/react-native-error-enospc-system-limit-for-number-of-file-watchers-reached
