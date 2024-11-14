package dolo

func (c *Client) SetBasicAuth(username, password string) {
	c.username = username
	c.password = password
}

func (c *Client) requiresBasicAuth() bool {
	return c.username != "" && c.password != ""
}
