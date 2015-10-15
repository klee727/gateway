package gateway

/**
* @brief Agent的接口，目前只支持http实现
 */
type Configurable interface {
	// 以map[ConfigId]md5sum形式列出agent上目前的配置项
	ListConfig() (map[string]string, error)
	// 将 configId, jsonConfig 配置下去
	DoConfig(configId string, jsonConfig string) error
	UnConfig(configId string) error
	Ping() error
}
