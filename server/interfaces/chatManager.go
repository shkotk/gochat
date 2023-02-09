package interfaces

type ChatManager interface {
	// Creates new Chat with specified chat name.
	Create(chatName string) error

	// Lists all existing chats.
	List() (chatNames []string, err error)

	// Adds provided client to chat with provided chat name.
	AddClient(client Client, chatName string) error
}
